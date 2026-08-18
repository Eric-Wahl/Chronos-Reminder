package com.chronos.reminder.planner.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class PlannerItemsResponse(
    val items: List<PlannerItemDto> = emptyList(),
)

@Serializable
data class PlannerItemDto(
    val id: String,
    val content: String = "",
    val checked: Boolean = false,
    val position: Int = 0,
    val period: String = "morning", // "morning" | "afternoon"
    @SerialName("dfm_item_id") val dfmItemId: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
)

@Serializable
data class AddPlannerItemRequest(
    val content: String,
    val period: String,
    @SerialName("dfm_item_id") val dfmItemId: String? = null,
)

@Serializable
data class UpdatePlannerItemRequest(
    val content: String? = null,
    val checked: Boolean? = null,
    val period: String? = null,
)

@Serializable
data class ReorderPlannerItemInput(
    val id: String,
    val position: Int,
    val period: String,
)

@Serializable
data class ReorderPlannerRequest(
    val items: List<ReorderPlannerItemInput>,
)
