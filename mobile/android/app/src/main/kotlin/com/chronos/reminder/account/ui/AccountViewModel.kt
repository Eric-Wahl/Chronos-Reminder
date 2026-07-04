package com.chronos.reminder.account.ui

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.chronos.reminder.account.data.AccountDto
import com.chronos.reminder.account.data.AccountRepository
import com.chronos.reminder.account.data.ApiKeyDto
import com.chronos.reminder.account.data.TimezoneDto
import com.chronos.reminder.auth.data.AuthRepository
import com.chronos.reminder.core.network.ApiResult
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class AccountUiState(
    val loading: Boolean = false,
    val error: String? = null,
    val successMessage: String? = null,
    val timezones: List<TimezoneDto> = emptyList(),
    val accountDeleted: Boolean = false,
    // API keys
    val apiKeys: List<ApiKeyDto> = emptyList(),
    val createdKey: ApiKeyDto? = null, // full key shown once
    // Merge state
    val pendingMergeToken: String? = null,
    val pendingMergeDiscordUsername: String? = null,
    // Email verification resend
    val resendingSent: Boolean = false,
)

@HiltViewModel
class AccountViewModel @Inject constructor(
    private val repository: AccountRepository,
    private val authRepository: AuthRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(AccountUiState())
    val state: StateFlow<AccountUiState> = _state.asStateFlow()

    val account: StateFlow<AccountDto?> = repository.account

    init {
        viewModelScope.launch {
            _state.update { it.copy(loading = true) }
            repository.refreshAccount()
            _state.update { it.copy(loading = false) }
        }
    }

    fun loadTimezones() {
        if (_state.value.timezones.isNotEmpty()) return
        viewModelScope.launch {
            val result = repository.getTimezones()
            if (result is ApiResult.Success) {
                _state.update { it.copy(timezones = result.data) }
            }
        }
    }

    fun updateTimezone(iana: String, successMessage: String) =
        runOp(successMessage) { repository.updateTimezone(iana) }

    fun updateDiscordSendImagePreference(enabled: Boolean, successMessage: String) =
        runOp(successMessage) { repository.updateDiscordSendImagePreference(enabled) }

    fun updateDiscordEnableSnoozePreference(enabled: Boolean, successMessage: String) =
        runOp(successMessage) { repository.updateDiscordEnableSnoozePreference(enabled) }

    fun updateUsername(username: String, successMessage: String) =
        runOp(successMessage) { repository.updateUsername(username.trim()) }

    fun updateEmail(email: String, successMessage: String) =
        runOp(successMessage) { repository.updateEmail(email.trim()) }

    fun changePassword(current: String, new: String, successMessage: String) =
        runOp(successMessage) { repository.changePassword(current, new) }

    fun linkDiscord(code: String, successMessage: String) {
        viewModelScope.launch {
            _state.update { it.copy(loading = true, error = null, successMessage = null) }
            when (val result = repository.linkDiscord(code)) {
                is ApiResult.Success -> {
                    val data = result.data
                    if (data.mergeRequired) {
                        _state.update {
                            it.copy(
                                loading = false,
                                pendingMergeToken = data.mergeToken,
                                pendingMergeDiscordUsername = data.discordUsername,
                            )
                        }
                    } else {
                        _state.update { it.copy(loading = false, successMessage = successMessage) }
                    }
                }
                is ApiResult.Error -> _state.update { it.copy(loading = false, error = result.message) }
                is ApiResult.NetworkError -> _state.update { it.copy(loading = false, error = "No internet connection") }
            }
        }
    }

    fun confirmMerge(successMessage: String) {
        val token = _state.value.pendingMergeToken ?: return
        _state.update { it.copy(pendingMergeToken = null, pendingMergeDiscordUsername = null) }
        runOp(successMessage) { repository.mergeAccounts(token) }
    }

    fun dismissMerge() = _state.update { it.copy(pendingMergeToken = null, pendingMergeDiscordUsername = null) }

    fun addAppIdentity(email: String, username: String, password: String, successMessage: String) =
        runOp(successMessage) { repository.addAppIdentity(email.trim(), username.trim(), password) }

    fun deleteAccount() {
        viewModelScope.launch {
            _state.update { it.copy(loading = true) }
            when (val result = repository.deleteAccount()) {
                is ApiResult.Success -> _state.update { it.copy(loading = false, accountDeleted = true) }
                is ApiResult.Error -> _state.update { it.copy(loading = false, error = result.message) }
                is ApiResult.NetworkError -> _state.update {
                    it.copy(loading = false, error = "No internet connection")
                }
            }
        }
    }

    // --- API keys ---

    fun loadApiKeys() {
        viewModelScope.launch {
            when (val result = repository.getApiKeys()) {
                is ApiResult.Success -> _state.update { it.copy(apiKeys = result.data) }
                is ApiResult.Error -> _state.update { it.copy(error = result.message) }
                is ApiResult.NetworkError -> _state.update { it.copy(error = "No internet connection") }
            }
        }
    }

    fun createApiKey(name: String) {
        viewModelScope.launch {
            when (val result = repository.createApiKey(name.trim())) {
                is ApiResult.Success -> {
                    _state.update { it.copy(createdKey = result.data) }
                    loadApiKeys()
                }
                is ApiResult.Error -> _state.update { it.copy(error = result.message) }
                is ApiResult.NetworkError -> _state.update { it.copy(error = "No internet connection") }
            }
        }
    }

    fun revokeApiKey(id: String) {
        viewModelScope.launch {
            when (val result = repository.revokeApiKey(id)) {
                is ApiResult.Success -> loadApiKeys()
                is ApiResult.Error -> _state.update { it.copy(error = result.message) }
                is ApiResult.NetworkError -> _state.update { it.copy(error = "No internet connection") }
            }
        }
    }

    fun dismissCreatedKey() = _state.update { it.copy(createdKey = null) }

    fun resendVerificationEmail(email: String, sentMessage: String) {
        viewModelScope.launch {
            when (authRepository.resendVerification(email)) {
                is ApiResult.Success -> _state.update { it.copy(resendingSent = true, successMessage = sentMessage) }
                else -> _state.update { it.copy(resendingSent = false) }
            }
        }
    }

    fun clearMessages() = _state.update { it.copy(error = null, successMessage = null) }

    private fun runOp(successMessage: String, op: suspend () -> ApiResult<Unit>) {
        viewModelScope.launch {
            _state.update { it.copy(loading = true, error = null, successMessage = null) }
            when (val result = op()) {
                is ApiResult.Success -> _state.update {
                    it.copy(loading = false, successMessage = successMessage)
                }
                is ApiResult.Error -> _state.update { it.copy(loading = false, error = result.message) }
                is ApiResult.NetworkError -> _state.update {
                    it.copy(loading = false, error = "No internet connection")
                }
            }
        }
    }
}
