package com.chronos.reminder.reminders.ui.create

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.chronos.reminder.account.data.AccountRepository
import com.chronos.reminder.core.MAX_REMINDERS_PER_ACCOUNT
import com.chronos.reminder.core.network.ApiResult
import com.chronos.reminder.core.storage.DestinationPreferencesStore
import com.chronos.reminder.reminders.data.ChannelDto
import com.chronos.reminder.reminders.data.DiscordApi
import com.chronos.reminder.reminders.data.GuildChannelsRequest
import com.chronos.reminder.reminders.data.GuildDto
import com.chronos.reminder.reminders.data.GuildsRequest
import com.chronos.reminder.reminders.data.RemindersRepository
import com.chronos.reminder.reminders.domain.Destination
import com.chronos.reminder.reminders.domain.RecurrenceType
import com.chronos.reminder.core.network.safeApiCall
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.time.LocalDate
import java.time.LocalTime
import java.time.ZoneId
import java.time.ZonedDateTime
import java.time.temporal.ChronoUnit
import javax.inject.Inject

data class CreateReminderUiState(
    val step: Int = 1, // 1 = when, 2 = what, 3 = where
    val form: ReminderForm = ReminderForm(),
    val submitting: Boolean = false,
    val created: Boolean = false,
    val error: String? = null,
    // Identity-driven destination availability
    val hasDiscordIdentity: Boolean = false,
    val hasEmailIdentity: Boolean = false,
    val accountEmail: String? = null,
    val discordUserId: String? = null,
    val accountId: String? = null,
    // Discord channel picker
    val guilds: List<GuildDto> = emptyList(),
    val channels: List<ChannelDto> = emptyList(),
    val loadingGuilds: Boolean = false,
    val loadingChannels: Boolean = false,
    val userTimezone: String = java.util.TimeZone.getDefault().id,
    val reminderLimitReached: Boolean = false,
)

@HiltViewModel
class CreateReminderViewModel @Inject constructor(
    private val remindersRepository: RemindersRepository,
    private val accountRepository: AccountRepository,
    private val discordApi: DiscordApi,
    private val destinationPrefs: DestinationPreferencesStore,
) : ViewModel() {

    private val _uiState = MutableStateFlow(
        CreateReminderUiState(
            form = ReminderForm(
                date = LocalDate.now(),
                time = LocalTime.now().plusMinutes(10).truncatedTo(ChronoUnit.MINUTES),
            ),
        ),
    )
    val uiState: StateFlow<CreateReminderUiState> = _uiState.asStateFlow()

    init {
        viewModelScope.launch {
            if (accountRepository.account.value == null) {
                accountRepository.refreshAccount()
            }
            val account = accountRepository.account.value
            val discord = account?.identities?.firstOrNull { it.provider == "discord" }
            val hasDiscord = discord != null
            val hasEmail = account?.email != null
            val email = account?.email
            val resolvedTimezone = accountRepository.userTimezone

            _uiState.update {
                it.copy(
                    hasDiscordIdentity = hasDiscord,
                    hasEmailIdentity = hasEmail,
                    accountEmail = email,
                    discordUserId = discord?.externalId,
                    accountId = account?.id,
                    userTimezone = resolvedTimezone,
                )
            }

            // The initial date/time were suggested using the device's
            // timezone (before the account's was known). Re-derive them
            // from the account's configured timezone now, but only if the
            // user hasn't started filling out the form yet.
            val zone = runCatching { ZoneId.of(resolvedTimezone) }.getOrDefault(ZoneId.systemDefault())
            val suggested = ZonedDateTime.now(zone).plusMinutes(10).truncatedTo(ChronoUnit.MINUTES)
            updateForm { form ->
                if (form.message.isEmpty() && form.destinations.isEmpty()) {
                    form.copy(date = suggested.toLocalDate(), time = suggested.toLocalTime())
                } else {
                    form
                }
            }

            // Pre-select the last destinations the user picked, if still available
            val lastDestinations = destinationPrefs.getLast()
            val preSelected = mutableListOf<FormDestination>()
            for (type in lastDestinations) {
                when (type) {
                    Destination.TYPE_DISCORD_DM -> if (hasDiscord && discord?.externalId != null)
                        preSelected.add(FormDestination(Destination.TYPE_DISCORD_DM, mapOf("user_id" to discord.externalId!!)))
                    Destination.TYPE_EMAIL -> if (hasEmail && email != null)
                        preSelected.add(FormDestination(Destination.TYPE_EMAIL, mapOf("email" to email), detail = email))
                    Destination.TYPE_ANDROID_PUSH -> account?.id?.let { id ->
                        preSelected.add(FormDestination(Destination.TYPE_ANDROID_PUSH, mapOf("account_id" to id)))
                    }
                }
            }
            if (preSelected.isNotEmpty()) {
                updateForm { it.copy(destinations = preSelected) }
            }

            val currentReminderCount = remindersRepository.getReminders().first().size
            _uiState.update {
                it.copy(reminderLimitReached = currentReminderCount >= MAX_REMINDERS_PER_ACCOUNT)
            }
        }
    }

    fun initFromForm(form: ReminderForm) {
        _uiState.update { it.copy(form = form) }
    }

    fun setDate(date: LocalDate) = updateForm { it.copy(date = date) }
    fun setTime(time: LocalTime) = updateForm { it.copy(time = time) }
    fun setRecurrence(recurrence: RecurrenceType) = updateForm { it.copy(recurrence = recurrence) }

    fun setMessage(message: String) =
        updateForm { it.copy(message = message.take(MAX_MESSAGE_LENGTH)) }

    fun addDestination(destination: FormDestination) {
        // No duplicate destination types except webhook/channel which can differ by target
        updateForm { form ->
            val duplicate = form.destinations.any {
                it.type == destination.type && it.metadata == destination.metadata
            }
            if (duplicate) form else form.copy(destinations = form.destinations + destination)
        }
    }

    fun addDiscordDmDestination() {
        val userId = _uiState.value.discordUserId ?: return
        addDestination(FormDestination(Destination.TYPE_DISCORD_DM, mapOf("user_id" to userId)))
    }

    fun addEmailDestination(email: String) {
        addDestination(FormDestination(Destination.TYPE_EMAIL, mapOf("email" to email), detail = email))
    }

    fun addPushDestination() {
        val accountId = _uiState.value.accountId ?: return
        addDestination(FormDestination(Destination.TYPE_ANDROID_PUSH, mapOf("account_id" to accountId)))
    }

    fun addChannelDestination(guild: GuildDto, channel: ChannelDto) {
        addDestination(
            FormDestination(
                type = Destination.TYPE_DISCORD_CHANNEL,
                metadata = mapOf("guild_id" to guild.id, "channel_id" to channel.id),
                detail = "${guild.name} #${channel.name}",
            ),
        )
    }

    fun addWebhookDestination(url: String, platform: String) {
        addDestination(
            FormDestination(
                type = Destination.TYPE_WEBHOOK,
                metadata = mapOf("url" to url, "platform" to platform),
                detail = url,
            ),
        )
    }

    fun removeDestination(destination: FormDestination) {
        updateForm { it.copy(destinations = it.destinations - destination) }
    }

    fun goToStep(step: Int) {
        _uiState.update { it.copy(step = step.coerceIn(1, 3), error = null) }
    }

    fun loadGuilds() {
        val accountId = _uiState.value.accountId ?: return
        if (_uiState.value.guilds.isNotEmpty()) return
        viewModelScope.launch {
            _uiState.update { it.copy(loadingGuilds = true) }
            val result = safeApiCall { discordApi.getGuilds(GuildsRequest(accountId)) }
            _uiState.update {
                it.copy(
                    loadingGuilds = false,
                    guilds = (result as? ApiResult.Success)?.data?.guilds ?: emptyList(),
                )
            }
        }
    }

    fun loadChannels(guildId: String) {
        val accountId = _uiState.value.accountId ?: return
        viewModelScope.launch {
            _uiState.update { it.copy(loadingChannels = true, channels = emptyList()) }
            val result = safeApiCall { discordApi.getGuildChannels(GuildChannelsRequest(accountId, guildId)) }
            _uiState.update {
                it.copy(
                    loadingChannels = false,
                    channels = (result as? ApiResult.Success)?.data?.channels ?: emptyList(),
                )
            }
        }
    }

    fun submit() {
        val state = _uiState.value
        if (state.form.destinations.isEmpty() || state.submitting) return
        if (state.reminderLimitReached) {
            _uiState.update {
                it.copy(error = "You have reached the maximum of $MAX_REMINDERS_PER_ACCOUNT reminders")
            }
            return
        }
        viewModelScope.launch {
            _uiState.update { it.copy(submitting = true, error = null) }
            when (val result = remindersRepository.createReminder(state.form.toRequest())) {
                is ApiResult.Success -> {
                    destinationPrefs.save(state.form.destinations.map { it.type })
                    _uiState.update { it.copy(submitting = false, created = true) }
                }
                is ApiResult.Error -> _uiState.update { it.copy(submitting = false, error = result.message) }
                is ApiResult.NetworkError -> _uiState.update {
                    it.copy(submitting = false, error = "No internet connection")
                }
            }
        }
    }

    fun clearError() = _uiState.update { it.copy(error = null) }

    private fun updateForm(transform: (ReminderForm) -> ReminderForm) {
        _uiState.update { it.copy(form = transform(it.form)) }
    }
}
