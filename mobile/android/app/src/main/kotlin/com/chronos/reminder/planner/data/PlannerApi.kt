package com.chronos.reminder.planner.data

import com.chronos.reminder.reminders.data.MessageResponse
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path

interface PlannerApi {

    @GET("api/planner/items")
    suspend fun getItems(): Response<PlannerItemsResponse>

    @POST("api/planner/items")
    suspend fun addItem(@Body body: AddPlannerItemRequest): Response<PlannerItemDto>

    @PUT("api/planner/items/{id}")
    suspend fun updateItem(@Path("id") id: String, @Body body: UpdatePlannerItemRequest): Response<PlannerItemDto>

    @DELETE("api/planner/items/{id}")
    suspend fun deleteItem(@Path("id") id: String): Response<MessageResponse>

    @PUT("api/planner/reorder")
    suspend fun reorder(@Body body: ReorderPlannerRequest): Response<PlannerItemsResponse>

    @DELETE("api/planner/items")
    suspend fun clearAll(): Response<MessageResponse>
}
