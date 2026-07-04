package com.chronos.reminder.dfm.ui

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.chronos.reminder.account.data.AccountRepository
import com.chronos.reminder.core.MAX_DFM_ITEMS_PER_NOTE
import com.chronos.reminder.core.network.ApiResult
import com.chronos.reminder.dfm.data.DfmItem
import com.chronos.reminder.dfm.data.DfmReminderInfo
import com.chronos.reminder.dfm.data.DfmReminderRequest
import com.chronos.reminder.dfm.data.DfmRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.time.LocalDate
import java.time.LocalTime
import java.time.format.DateTimeFormatter
import javax.inject.Inject

data class DfmUiState(
    val refreshing: Boolean = false,
    val error: String? = null,
    val noteSent: Boolean = false,
    val itemAdded: Boolean = false,
    val sendingNow: Boolean = false,
    val hasDiscordIdentity: Boolean = false,
    val hasEmailIdentity: Boolean = false,
    val userTimezone: String = java.util.TimeZone.getDefault().id,
)

@HiltViewModel
class DfmViewModel @Inject constructor(
    private val repository: DfmRepository,
    private val accountRepository: AccountRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(DfmUiState())
    val state: StateFlow<DfmUiState> = _state.asStateFlow()

    val items: StateFlow<List<DfmItem>> = repository.getItems()
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), emptyList())

    val reminderInfo: StateFlow<DfmReminderInfo?> = repository.reminderInfo

    init {
        refresh()
        viewModelScope.launch {
            if (accountRepository.account.value == null) {
                accountRepository.refreshAccount()
            }
            val account = accountRepository.account.value
            val identities = account?.identities.orEmpty()
            _state.update {
                it.copy(
                    hasDiscordIdentity = identities.any { id -> id.provider == "discord" },
                    hasEmailIdentity = account?.email != null,
                    userTimezone = accountRepository.userTimezone,
                )
            }
        }
    }

    fun refresh() {
        viewModelScope.launch {
            _state.update { it.copy(refreshing = true) }
            val result = repository.refresh()
            _state.update { it.copy(refreshing = false, error = result.errorOrNull()) }
        }
    }

    fun addItem(content: String) {
        if (content.isBlank()) return
        if (items.value.size >= MAX_DFM_ITEMS_PER_NOTE) {
            _state.update { it.copy(error = "You have reached the maximum of $MAX_DFM_ITEMS_PER_NOTE items") }
            return
        }
        viewModelScope.launch {
            val result = repository.addItem(content.trim())
            _state.update { it.copy(error = result.errorOrNull(), itemAdded = result.errorOrNull() == null) }
        }
    }

    fun consumeItemAdded() = _state.update { it.copy(itemAdded = false) }

    fun toggleItem(item: DfmItem, checked: Boolean) {
        viewModelScope.launch {
            val result = repository.setItemChecked(item, checked)
            _state.update { it.copy(error = result.errorOrNull()) }
        }
    }

    fun editItem(item: DfmItem, content: String) {
        if (content.isBlank() || content == item.content) return
        viewModelScope.launch {
            val result = repository.updateItem(item.id, content = content.trim())
            _state.update { it.copy(error = result.errorOrNull()) }
        }
    }

    fun deleteItem(id: String) {
        viewModelScope.launch {
            val result = repository.deleteItem(id)
            _state.update { it.copy(error = result.errorOrNull()) }
        }
    }

    fun setReminder(date: LocalDate, time: LocalTime, recurrence: String, destinations: List<String>) {
        viewModelScope.launch {
            val result = repository.setReminder(
                DfmReminderRequest(
                    date = date.format(DateTimeFormatter.ISO_LOCAL_DATE),
                    time = time.format(DateTimeFormatter.ofPattern("HH:mm")),
                    recurrence = recurrence,
                    destinations = destinations,
                ),
            )
            _state.update { it.copy(error = result.errorOrNull()) }
        }
    }

    fun removeReminder() {
        viewModelScope.launch {
            val result = repository.removeReminder()
            _state.update { it.copy(error = result.errorOrNull()) }
        }
    }

    fun sendNow() {
        viewModelScope.launch {
            _state.update { it.copy(sendingNow = true) }
            val result = repository.sendNow()
            _state.update {
                it.copy(
                    sendingNow = false,
                    noteSent = result is ApiResult.Success,
                    error = result.errorOrNull(),
                )
            }
        }
    }

    fun consumeNoteSent() = _state.update { it.copy(noteSent = false) }

    fun clearError() = _state.update { it.copy(error = null) }

    private fun <T> ApiResult<T>.errorOrNull(): String? = when (this) {
        is ApiResult.Success -> null
        is ApiResult.Error -> message
        is ApiResult.NetworkError -> "No internet connection"
    }
}
