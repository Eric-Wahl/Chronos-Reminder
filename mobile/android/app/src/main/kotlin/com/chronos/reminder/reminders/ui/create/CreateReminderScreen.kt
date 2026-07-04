package com.chronos.reminder.reminders.ui.create

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.chronos.reminder.R
import com.chronos.reminder.core.ui.components.ChronosTopBar
import com.chronos.reminder.core.ui.components.ErrorBanner
import com.chronos.reminder.core.ui.theme.AccentOrange
import com.chronos.reminder.core.ui.theme.BackgroundMain
import com.chronos.reminder.core.ui.theme.BackgroundMuted

private const val TOTAL_STEPS = 3

@Composable
fun CreateReminderScreen(
    onCreated: () -> Unit,
    onBack: () -> Unit,
    viewModel: CreateReminderViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    var dateError by rememberSaveable { mutableStateOf<String?>(null) }
    val dateInPastError = stringResource(R.string.error_date_in_past)
    val reminderLimitReachedError = stringResource(
        R.string.reminder_limit_reached,
        com.chronos.reminder.core.MAX_REMINDERS_PER_ACCOUNT,
    )

    LaunchedEffect(uiState.created) {
        if (uiState.created) onCreated()
    }

    Scaffold(
        containerColor = BackgroundMain,
        topBar = {
            ChronosTopBar(
                title = stringResource(R.string.create_reminder_title),
                onBack = {
                    if (uiState.step > 1) viewModel.goToStep(uiState.step - 1) else onBack()
                },
            )
        },
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 16.dp),
            ) {
                LinearProgressIndicator(
                    progress = { uiState.step / TOTAL_STEPS.toFloat() },
                    modifier = Modifier.fillMaxWidth(),
                    color = AccentOrange,
                    trackColor = BackgroundMuted,
                )
                Spacer(Modifier.height(24.dp))

                when (uiState.step) {
                    1 -> {
                        WhenStep(
                            form = uiState.form,
                            onDateSelected = {
                                dateError = null
                                viewModel.setDate(it)
                            },
                            onTimeSelected = {
                                dateError = null
                                viewModel.setTime(it)
                            },
                            onRecurrenceSelected = viewModel::setRecurrence,
                        )
                        Spacer(Modifier.height(32.dp))
                        StepNavButtons(
                            showBack = false,
                            nextLabel = stringResource(R.string.next),
                            onBack = {},
                            onNext = {
                                if (uiState.form.isDateTimeInFuture(uiState.userTimezone)) {
                                    viewModel.goToStep(2)
                                } else {
                                    dateError = dateInPastError
                                }
                            },
                            nextEnabled = uiState.form.date != null && uiState.form.time != null,
                        )
                    }

                    2 -> {
                        WhatStep(form = uiState.form, onMessageChange = viewModel::setMessage)
                        Spacer(Modifier.height(32.dp))
                        StepNavButtons(
                            showBack = true,
                            nextLabel = stringResource(R.string.next),
                            onBack = { viewModel.goToStep(1) },
                            onNext = { viewModel.goToStep(3) },
                            nextEnabled = uiState.form.message.isNotBlank(),
                        )
                    }

                    else -> {
                        WhereStep(
                            uiState = uiState,
                            onAddDiscordDm = viewModel::addDiscordDmDestination,
                            onAddEmail = viewModel::addEmailDestination,
                            onAddPush = viewModel::addPushDestination,
                            onAddChannel = viewModel::addChannelDestination,
                            onAddWebhook = viewModel::addWebhookDestination,
                            onRemove = viewModel::removeDestination,
                            onLoadGuilds = viewModel::loadGuilds,
                            onLoadChannels = viewModel::loadChannels,
                        )
                        Spacer(Modifier.height(32.dp))
                        StepNavButtons(
                            showBack = true,
                            nextLabel = stringResource(R.string.create),
                            onBack = { viewModel.goToStep(2) },
                            onNext = viewModel::submit,
                            nextEnabled = uiState.form.destinations.isNotEmpty() &&
                                !uiState.reminderLimitReached,
                            nextLoading = uiState.submitting,
                        )
                    }
                }
                Spacer(Modifier.height(24.dp))
            }

            ErrorBanner(
                message = dateError
                    ?: uiState.error
                    ?: reminderLimitReachedError.takeIf { uiState.reminderLimitReached },
                modifier = Modifier.align(Alignment.BottomCenter),
                onDismiss = {
                    dateError = null
                    viewModel.clearError()
                },
            )
        }
    }
}
