package com.chronos.reminder.account.data

import com.chronos.reminder.reminders.data.MessageResponse
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path

interface AccountApi {

    @GET("api/account")
    suspend fun getAccount(): Response<AccountDto>

    @PUT("api/account/timezone")
    suspend fun updateTimezone(@Body body: TimezoneRequest): Response<MessageResponse>

    @PUT("api/account/preferences")
    suspend fun updatePreferences(@Body body: PreferencesRequest): Response<MessageResponse>

    @PUT("api/account/identity/app/username")
    suspend fun updateUsername(@Body body: UsernameRequest): Response<MessageResponse>

    @PUT("api/account/identity/app/email")
    suspend fun updateEmail(@Body body: EmailRequest): Response<MessageResponse>

    @POST("api/account/identity/app/change-password")
    suspend fun changePassword(@Body body: ChangePasswordRequest): Response<MessageResponse>

    @DELETE("api/account")
    suspend fun deleteAccount(): Response<MessageResponse>

    @POST("api/account/identity/mobile")
    suspend fun ensureMobileIdentity(@Body body: MobileIdentityRequest): Response<IdentityDto>

    @POST("api/account/identity/discord/link")
    suspend fun linkDiscord(@Body body: DiscordLinkRequest): Response<LinkDiscordResponse>

    @POST("api/account/merge")
    suspend fun mergeAccounts(@Body body: MergeRequest): Response<MessageResponse>

    @POST("api/account/identity/app")
    suspend fun addAppIdentity(@Body body: AddAppIdentityRequest): Response<IdentityDto>

    @GET("api/timezones")
    suspend fun getTimezones(): Response<List<TimezoneDto>>

    @GET("api/health")
    suspend fun getHealth(): Response<HealthResponse>

    @GET("api/api-keys")
    suspend fun getApiKeys(): Response<ApiKeysListResponse>

    @POST("api/api-keys")
    suspend fun createApiKey(@Body body: CreateApiKeyRequest): Response<ApiKeyDto>

    @DELETE("api/api-keys/{id}")
    suspend fun revokeApiKey(@Path("id") id: String): Response<MessageResponse>
}
