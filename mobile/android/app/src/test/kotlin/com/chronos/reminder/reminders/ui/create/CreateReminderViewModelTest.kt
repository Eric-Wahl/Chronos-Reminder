package com.chronos.reminder.reminders.ui.create

import com.chronos.reminder.MainDispatcherRule
import com.chronos.reminder.account.data.AccountDto
import com.chronos.reminder.account.data.AccountRepository
import com.chronos.reminder.account.data.IdentityDto
import com.chronos.reminder.core.network.ApiResult
import com.chronos.reminder.core.storage.DestinationPreferencesStore
import com.chronos.reminder.reminders.data.DiscordApi
import com.chronos.reminder.reminders.data.RemindersRepository
import com.chronos.reminder.reminders.domain.Destination
import com.chronos.reminder.reminders.domain.RecurrenceType
import io.mockk.coEvery
import io.mockk.every
import io.mockk.mockk
import io.mockk.slot
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import java.time.LocalDate
import java.time.LocalTime

class CreateReminderViewModelTest {

    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    private val repository: RemindersRepository = mockk()
    private val accountRepository: AccountRepository = mockk()
    private val discordApi: DiscordApi = mockk()
    private val destinationPrefs: DestinationPreferencesStore = mockk()

    private val account = AccountDto(
        id = "acc-1",
        email = "me@example.com",
        identities = listOf(
            IdentityDto(provider = "discord", externalId = "disc-42", username = "me#1234"),
        ),
    )

    private lateinit var viewModel: CreateReminderViewModel

    @Before
    fun setUp() {
        every { accountRepository.account } returns MutableStateFlow(account)
        every { accountRepository.userTimezone } returns "Europe/Paris"
        every { destinationPrefs.getLast() } returns emptyList()
        every { destinationPrefs.save(any()) } returns Unit
        every { repository.getReminders() } returns flowOf(emptyList())
        viewModel = CreateReminderViewModel(repository, accountRepository, discordApi, destinationPrefs)
    }

    @Test
    fun `account identities populate destination availability`() = runTest {
        val state = viewModel.uiState.value
        assertTrue(state.hasDiscordIdentity)
        assertTrue(state.hasEmailIdentity)
        assertEquals("acc-1", state.accountId)
        assertEquals("disc-42", state.discordUserId)
    }

    @Test
    fun `step transitions are clamped to wizard range`() {
        viewModel.goToStep(2)
        assertEquals(2, viewModel.uiState.value.step)
        viewModel.goToStep(7)
        assertEquals(3, viewModel.uiState.value.step)
        viewModel.goToStep(0)
        assertEquals(1, viewModel.uiState.value.step)
    }

    @Test
    fun `duplicate destinations are not added twice`() {
        viewModel.addDiscordDmDestination()
        viewModel.addDiscordDmDestination()

        assertEquals(1, viewModel.uiState.value.form.destinations.size)
    }

    @Test
    fun `submit without destinations is rejected`() = runTest {
        viewModel.setDate(LocalDate.of(2030, 1, 1))
        viewModel.setTime(LocalTime.of(9, 0))
        viewModel.setMessage("hello")

        viewModel.submit()

        assertFalse(viewModel.uiState.value.created)
        assertFalse(viewModel.uiState.value.submitting)
    }

    @Test
    fun `submit builds request from form`() = runTest {
        val requestSlot = slot<com.chronos.reminder.reminders.data.CreateReminderRequest>()
        coEvery { repository.createReminder(capture(requestSlot)) } returns ApiResult.Success(
            mockk(relaxed = true),
        )

        viewModel.setDate(LocalDate.of(2030, 1, 15))
        viewModel.setTime(LocalTime.of(14, 30))
        viewModel.setRecurrence(RecurrenceType.WEEKLY)
        viewModel.setMessage("Water the plants")
        viewModel.addDiscordDmDestination()
        viewModel.addEmailDestination("me@example.com")
        viewModel.submit()

        val request = requestSlot.captured
        assertEquals("2030-01-15", request.date)
        assertEquals("14:30", request.time)
        assertEquals("WEEKLY", request.recurrence)
        assertEquals("Water the plants", request.message)
        assertEquals(2, request.destinations.size)
        assertEquals(Destination.TYPE_DISCORD_DM, request.destinations[0].type)
        assertTrue(viewModel.uiState.value.created)
    }

    @Test
    fun `message is capped at max length`() {
        viewModel.setMessage("x".repeat(MAX_MESSAGE_LENGTH + 100))
        assertEquals(MAX_MESSAGE_LENGTH, viewModel.uiState.value.form.message.length)
    }

    @Test
    fun `past date fails future validation`() {
        viewModel.setDate(LocalDate.of(2020, 1, 1))
        viewModel.setTime(LocalTime.NOON)
        assertFalse(viewModel.uiState.value.form.isDateTimeInFuture("Europe/Paris"))
    }
}
