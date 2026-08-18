package com.chronos.reminder.planner.ui

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.chronos.reminder.core.network.ApiResult
import com.chronos.reminder.dfm.data.DfmItem
import com.chronos.reminder.dfm.data.DfmRepository
import com.chronos.reminder.planner.data.PlannerItem
import com.chronos.reminder.planner.data.PlannerPeriod
import com.chronos.reminder.planner.data.PlannerRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class PlannerUiState(
    val refreshing: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class PlannerViewModel @Inject constructor(
    private val repository: PlannerRepository,
    private val dfmRepository: DfmRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(PlannerUiState())
    val state: StateFlow<PlannerUiState> = _state.asStateFlow()

    val items: StateFlow<List<PlannerItem>> = repository.items
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), emptyList())

    // Used to power the "link to a Don't Forget Me item" autocomplete
    val dfmItems: StateFlow<List<DfmItem>> = dfmRepository.getItems()
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), emptyList())

    init {
        refresh()
        viewModelScope.launch { dfmRepository.refresh() }
    }

    fun refresh() {
        viewModelScope.launch {
            _state.update { it.copy(refreshing = true) }
            val result = repository.refresh()
            _state.update { it.copy(refreshing = false, error = result.errorOrNull()) }
        }
    }

    fun addItem(content: String, period: PlannerPeriod, dfmItemId: String?) {
        if (content.isBlank()) return
        viewModelScope.launch {
            val result = repository.addItem(content.trim(), period, dfmItemId)
            _state.update { it.copy(error = result.errorOrNull()) }
        }
    }

    fun toggleChecked(item: PlannerItem) {
        viewModelScope.launch {
            val result = repository.setChecked(item, !item.checked)
            _state.update { it.copy(error = result.errorOrNull()) }
        }
    }

    fun movePeriod(item: PlannerItem, newPeriod: PlannerPeriod) {
        if (item.period == newPeriod) return
        viewModelScope.launch {
            val current = items.value
            val target = current.filter { it.period == newPeriod }
            val reordered = current.map {
                if (it.id == item.id) it.copy(period = newPeriod, position = target.size) else it
            }
            val result = repository.reorder(reordered)
            _state.update { it.copy(error = result.errorOrNull()) }
        }
    }

    fun reorderWithinPeriod(period: PlannerPeriod, newOrderIds: List<String>) {
        viewModelScope.launch {
            val current = items.value
            val reorderedColumn = newOrderIds.mapNotNull { id -> current.find { it.id == id } }
            val others = current.filter { it.period != period }
            val result = repository.reorder(others + reorderedColumn)
            _state.update { it.copy(error = result.errorOrNull()) }
        }
    }

    fun deleteItem(item: PlannerItem) {
        viewModelScope.launch {
            val result = repository.deleteItem(item.id)
            _state.update { it.copy(error = result.errorOrNull()) }
        }
    }

    fun clearAll() {
        viewModelScope.launch {
            val result = repository.clearAll()
            _state.update { it.copy(error = result.errorOrNull()) }
        }
    }

    fun clearError() = _state.update { it.copy(error = null) }

    private fun <T> ApiResult<T>.errorOrNull(): String? = when (this) {
        is ApiResult.Success -> null
        is ApiResult.Error -> message
        is ApiResult.NetworkError -> "No internet connection"
    }
}
