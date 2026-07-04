package com.chronos.reminder.account.data

import com.chronos.reminder.core.network.ApiResult
import com.chronos.reminder.core.network.safeApiCall
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AccountRepository @Inject constructor(
    private val api: AccountApi,
) {

    private val _account = MutableStateFlow<AccountDto?>(null)
    val account: StateFlow<AccountDto?> = _account.asStateFlow()

    private var cachedTimezones: List<TimezoneDto>? = null

    val userTimezone: String
        get() = _account.value?.timezone?.ianaLocation ?: java.util.TimeZone.getDefault().id

    suspend fun refreshAccount(): ApiResult<AccountDto> {
        val result = safeApiCall { api.getAccount() }
        if (result is ApiResult.Success) {
            _account.value = result.data
        }
        return result
    }

    suspend fun getTimezones(): ApiResult<List<TimezoneDto>> {
        cachedTimezones?.let { return ApiResult.Success(it) }
        val result = safeApiCall { api.getTimezones() }
        if (result is ApiResult.Success) {
            cachedTimezones = result.data
        }
        return result
    }

    suspend fun updateTimezone(iana: String): ApiResult<Unit> =
        safeApiCall { api.updateTimezone(TimezoneRequest(iana)) }
            .onSuccess { refreshAccount() }
            .map { }

    suspend fun updateDiscordSendImagePreference(enabled: Boolean): ApiResult<Unit> =
        safeApiCall { api.updatePreferences(PreferencesRequest(discordSendImage = enabled)) }
            .onSuccess { refreshAccount() }
            .map { }

    suspend fun updateDiscordEnableSnoozePreference(enabled: Boolean): ApiResult<Unit> =
        safeApiCall { api.updatePreferences(PreferencesRequest(discordEnableSnooze = enabled)) }
            .onSuccess { refreshAccount() }
            .map { }

    suspend fun updateUsername(username: String): ApiResult<Unit> =
        safeApiCall { api.updateUsername(UsernameRequest(username)) }
            .onSuccess { refreshAccount() }
            .map { }

    suspend fun updateEmail(email: String): ApiResult<Unit> =
        safeApiCall { api.updateEmail(EmailRequest(email)) }
            .onSuccess { refreshAccount() }
            .map { }

    suspend fun changePassword(current: String, new: String): ApiResult<Unit> =
        safeApiCall { api.changePassword(ChangePasswordRequest(current, new)) }.map { }

    suspend fun deleteAccount(): ApiResult<Unit> =
        safeApiCall { api.deleteAccount() }.map { }

    suspend fun linkDiscord(code: String): ApiResult<LinkDiscordResponse> =
        safeApiCall { api.linkDiscord(DiscordLinkRequest(code)) }
            .onSuccess { if (!it.mergeRequired) refreshAccount() }

    suspend fun mergeAccounts(mergeToken: String): ApiResult<Unit> =
        safeApiCall { api.mergeAccounts(MergeRequest(mergeToken)) }
            .onSuccess { refreshAccount() }
            .map { }

    suspend fun addAppIdentity(email: String, username: String, password: String): ApiResult<Unit> =
        safeApiCall { api.addAppIdentity(AddAppIdentityRequest(email, username, password)) }
            .onSuccess { refreshAccount() }
            .map { }

    suspend fun getApiKeys(): ApiResult<List<ApiKeyDto>> =
        safeApiCall { api.getApiKeys() }.map { it.keys }

    suspend fun createApiKey(name: String): ApiResult<ApiKeyDto> =
        safeApiCall { api.createApiKey(CreateApiKeyRequest(name)) }

    suspend fun revokeApiKey(id: String): ApiResult<Unit> =
        safeApiCall { api.revokeApiKey(id) }.map { }

    fun clear() {
        _account.value = null
    }
}
