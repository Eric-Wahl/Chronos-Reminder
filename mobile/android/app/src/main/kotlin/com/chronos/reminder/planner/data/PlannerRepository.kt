package com.chronos.reminder.planner.data

import com.chronos.reminder.core.network.ApiResult
import com.chronos.reminder.core.network.safeApiCall
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Singleton

enum class PlannerPeriod(val apiValue: String) {
    MORNING("morning"),
    AFTERNOON("afternoon");

    companion object {
        fun fromApi(value: String): PlannerPeriod =
            entries.firstOrNull { it.apiValue == value } ?: MORNING
    }
}

data class PlannerItem(
    val id: String,
    val content: String,
    val checked: Boolean,
    val position: Int,
    val period: PlannerPeriod,
    val dfmItemId: String?,
)

private fun PlannerItemDto.toDomain() = PlannerItem(
    id = id,
    content = content,
    checked = checked,
    position = position,
    period = PlannerPeriod.fromApi(period),
    dfmItemId = dfmItemId,
)

// This is a lightweight, web/mobile-only feature with no offline dispatch
// requirement, so it's kept as a simple in-memory StateFlow cache (unlike
// Reminders/DFM, which are Room-backed for background sync).
@Singleton
class PlannerRepository @Inject constructor(
    private val api: PlannerApi,
) {
    private val _items = MutableStateFlow<List<PlannerItem>>(emptyList())
    val items: StateFlow<List<PlannerItem>> = _items.asStateFlow()

    suspend fun refresh(): ApiResult<Unit> =
        safeApiCall { api.getItems() }.map { response ->
            _items.value = response.items.map { it.toDomain() }
        }

    suspend fun addItem(content: String, period: PlannerPeriod, dfmItemId: String? = null): ApiResult<PlannerItem> {
        val result = safeApiCall { api.addItem(AddPlannerItemRequest(content, period.apiValue, dfmItemId)) }
        if (result is ApiResult.Success) {
            val item = result.data.toDomain()
            _items.value = _items.value + item
        }
        return result.map { it.toDomain() }
    }

    // Optimistic checkbox toggle: update local state first, revert on API error
    suspend fun setChecked(item: PlannerItem, checked: Boolean): ApiResult<Unit> {
        _items.value = _items.value.map { if (it.id == item.id) it.copy(checked = checked) else it }
        val result = safeApiCall { api.updateItem(item.id, UpdatePlannerItemRequest(checked = checked)) }
        if (result !is ApiResult.Success) {
            _items.value = _items.value.map { if (it.id == item.id) it.copy(checked = item.checked) else it }
        }
        return result.map { }
    }

    suspend fun deleteItem(id: String): ApiResult<Unit> {
        val previous = _items.value
        _items.value = _items.value.filterNot { it.id == id }
        val result = safeApiCall { api.deleteItem(id) }
        if (result !is ApiResult.Success) {
            _items.value = previous
        }
        return result.map { }
    }

    // Persists a full reorder (position within each period, and the period
    // itself), e.g. after a drag-and-drop reorder or a move between columns.
    suspend fun reorder(allItems: List<PlannerItem>): ApiResult<Unit> {
        _items.value = allItems
        val payload = PlannerPeriod.entries.flatMap { period ->
            allItems.filter { it.period == period }
                .mapIndexed { index, item -> ReorderPlannerItemInput(item.id, index, period.apiValue) }
        }
        return safeApiCall { api.reorder(ReorderPlannerRequest(payload)) }.map { }
    }

    suspend fun clearAll(): ApiResult<Unit> {
        val previous = _items.value
        _items.value = emptyList()
        val result = safeApiCall { api.clearAll() }
        if (result !is ApiResult.Success) {
            _items.value = previous
        }
        return result.map { }
    }
}
