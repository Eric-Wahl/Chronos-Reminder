package com.chronos.reminder.dfm.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Checklist
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CheckboxDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.chronos.reminder.R
import com.chronos.reminder.core.ui.components.ChronosButton
import com.chronos.reminder.core.ui.components.ChronosButtonStyle
import com.chronos.reminder.core.ui.components.ChronosCard
import com.chronos.reminder.core.ui.components.ChronosTextField
import com.chronos.reminder.core.ui.components.ChronosTopBar
import com.chronos.reminder.core.ui.components.ConfirmDeleteDialog
import com.chronos.reminder.core.ui.components.EmptyState
import com.chronos.reminder.core.ui.components.ErrorBanner
import com.chronos.reminder.core.ui.theme.AccentOrange
import com.chronos.reminder.core.ui.theme.BackgroundCard
import com.chronos.reminder.core.ui.theme.BackgroundMain
import com.chronos.reminder.core.ui.theme.ForegroundMuted
import com.chronos.reminder.core.util.formatNextFire
import com.chronos.reminder.dfm.data.DfmItem
import com.chronos.reminder.reminders.domain.Destination
import com.chronos.reminder.reminders.domain.RecurrenceType
import com.chronos.reminder.reminders.ui.create.RecurrenceChipRow
import com.chronos.reminder.reminders.ui.create.WhenStep
import com.chronos.reminder.reminders.ui.create.ReminderForm
import java.time.LocalDate
import java.time.LocalTime
import java.time.ZoneId

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DfmScreen(viewModel: DfmViewModel = hiltViewModel()) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val items by viewModel.items.collectAsStateWithLifecycle()
    val itemLimitReached = items.size >= com.chronos.reminder.core.MAX_DFM_ITEMS_PER_NOTE
    val reminderInfo by viewModel.reminderInfo.collectAsStateWithLifecycle()

    var newItemText by rememberSaveable { mutableStateOf("") }
    var showReminderSheet by rememberSaveable { mutableStateOf(false) }
    var itemPendingDelete by rememberSaveable { mutableStateOf<String?>(null) }
    val snackbarHostState = remember { SnackbarHostState() }
    val noteSentText = stringResource(R.string.dfm_note_sent)
    val itemAddedText = stringResource(R.string.dfm_item_added)

    LaunchedEffect(state.noteSent) {
        if (state.noteSent) {
            snackbarHostState.showSnackbar(noteSentText)
            viewModel.consumeNoteSent()
        }
    }
    LaunchedEffect(state.itemAdded) {
        if (state.itemAdded) {
            snackbarHostState.showSnackbar(itemAddedText)
            viewModel.consumeItemAdded()
        }
    }

    Scaffold(
        containerColor = BackgroundMain,
        topBar = { ChronosTopBar(title = stringResource(R.string.dfm_title)) },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(horizontal = 16.dp),
            ) {
                val info = reminderInfo
                if (info != null) {
                    ChronosCard(modifier = Modifier.fillMaxWidth()) {
                        Column(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(12.dp),
                        ) {
                            Text(
                                text = stringResource(
                                    R.string.dfm_reminder_banner,
                                    info.recurrence,
                                    info.nextFireUtc?.let { formatNextFire(it, state.userTimezone) } ?: "—",
                                ),
                                style = MaterialTheme.typography.bodyMedium,
                            )
                            Spacer(Modifier.height(10.dp))
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                ChronosButton(
                                    text = stringResource(R.string.dfm_edit_reminder),
                                    onClick = { showReminderSheet = true },
                                    style = ChronosButtonStyle.Secondary,
                                    modifier = Modifier.weight(1f),
                                )
                                ChronosButton(
                                    text = stringResource(R.string.dfm_remove_reminder),
                                    onClick = viewModel::removeReminder,
                                    style = ChronosButtonStyle.Destructive,
                                    modifier = Modifier.weight(1f),
                                )
                            }
                        }
                    }
                } else {
                    ChronosButton(
                        text = stringResource(R.string.dfm_set_reminder),
                        onClick = { showReminderSheet = true },
                        modifier = Modifier.fillMaxWidth(),
                        style = ChronosButtonStyle.Secondary,
                    )
                    Spacer(Modifier.height(6.dp))
                    Text(
                        text = stringResource(R.string.dfm_description),
                        style = MaterialTheme.typography.bodySmall,
                        color = ForegroundMuted,
                        textAlign = TextAlign.Center,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 4.dp),
                    )
                }
                Spacer(Modifier.height(12.dp))

                Row(verticalAlignment = Alignment.CenterVertically) {
                    ChronosTextField(
                        value = newItemText,
                        onValueChange = { newItemText = it },
                        modifier = Modifier.weight(1f),
                        placeholder = stringResource(R.string.dfm_add_item_hint),
                        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                        keyboardActions = KeyboardActions(onDone = {
                            if (newItemText.isNotBlank() && !itemLimitReached) {
                                viewModel.addItem(newItemText)
                                newItemText = ""
                            }
                        }),
                        enabled = !itemLimitReached,
                    )
                    Spacer(Modifier.width(8.dp))
                    IconButton(
                        onClick = {
                            viewModel.addItem(newItemText)
                            newItemText = ""
                        },
                        enabled = newItemText.isNotBlank() && !itemLimitReached,
                    ) {
                        Icon(
                            Icons.Default.Add,
                            contentDescription = stringResource(R.string.dfm_add_item),
                            tint = AccentOrange,
                        )
                    }
                }
                if (itemLimitReached) {
                    Text(
                        text = stringResource(
                            R.string.dfm_item_limit_reached,
                            com.chronos.reminder.core.MAX_DFM_ITEMS_PER_NOTE,
                        ),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                }
                Spacer(Modifier.height(8.dp))

                if (items.isEmpty()) {
                    Box(modifier = Modifier.weight(1f)) {
                        EmptyState(
                            icon = Icons.Default.Checklist,
                            title = stringResource(R.string.dfm_empty_title),
                            subtitle = stringResource(R.string.dfm_empty_subtitle),
                        )
                    }
                } else {
                    LazyColumn(
                        modifier = Modifier.weight(1f),
                        verticalArrangement = Arrangement.spacedBy(4.dp),
                    ) {
                        items(items, key = { it.id }) { item ->
                            DfmItemRow(
                                item = item,
                                onToggle = { checked -> viewModel.toggleItem(item, checked) },
                                onEdit = { content -> viewModel.editItem(item, content) },
                                onDelete = { itemPendingDelete = item.id },
                            )
                        }
                    }
                }

                ChronosButton(
                    text = stringResource(R.string.dfm_send_now),
                    onClick = viewModel::sendNow,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 12.dp),
                    loading = state.sendingNow,
                    enabled = items.isNotEmpty(),
                )
            }

            ErrorBanner(
                message = state.error,
                modifier = Modifier.align(Alignment.BottomCenter),
                onDismiss = viewModel::clearError,
            )
        }
    }

    if (itemPendingDelete != null) {
        ConfirmDeleteDialog(
            title = stringResource(R.string.dfm_delete_confirm_title),
            text = stringResource(R.string.dfm_delete_confirm_text),
            onConfirm = {
                viewModel.deleteItem(itemPendingDelete!!)
                itemPendingDelete = null
            },
            onDismiss = { itemPendingDelete = null },
        )
    }

    if (showReminderSheet) {
        ModalBottomSheet(onDismissRequest = { showReminderSheet = false }, containerColor = BackgroundCard) {
            DfmReminderSheetContent(
                hasDiscordIdentity = state.hasDiscordIdentity,
                hasEmailIdentity = state.hasEmailIdentity,
                userTimezone = state.userTimezone,
                onConfirm = { date, time, recurrence, destinations ->
                    viewModel.setReminder(date, time, recurrence, destinations)
                    showReminderSheet = false
                },
            )
        }
    }
}

@Composable
private fun DfmItemRow(
    item: DfmItem,
    onToggle: (Boolean) -> Unit,
    onEdit: (String) -> Unit,
    onDelete: () -> Unit,
) {
    var editing by rememberSaveable(item.id) { mutableStateOf(false) }
    var editText by rememberSaveable(item.id) { mutableStateOf(item.content) }

    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Checkbox(
            checked = item.checked,
            onCheckedChange = onToggle,
            colors = CheckboxDefaults.colors(checkedColor = AccentOrange),
        )
        if (editing) {
            ChronosTextField(
                value = editText,
                onValueChange = { editText = it },
                modifier = Modifier.weight(1f),
                keyboardActions = androidx.compose.foundation.text.KeyboardActions(
                    onDone = {
                        onEdit(editText)
                        editing = false
                    },
                ),
                keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(
                    imeAction = androidx.compose.ui.text.input.ImeAction.Done,
                ),
            )
        } else {
            Text(
                text = item.content,
                style = MaterialTheme.typography.bodyLarge.copy(
                    textDecoration = if (item.checked) TextDecoration.LineThrough else TextDecoration.None,
                ),
                color = if (item.checked) ForegroundMuted else MaterialTheme.colorScheme.onBackground,
                modifier = Modifier
                    .weight(1f)
                    .clickable {
                        editText = item.content
                        editing = true
                    },
            )
        }
        IconButton(onClick = onDelete) {
            Icon(
                Icons.Default.Close,
                contentDescription = stringResource(R.string.dfm_delete_item),
                tint = ForegroundMuted,
            )
        }
    }
}

@Composable
private fun DfmReminderSheetContent(
    hasDiscordIdentity: Boolean,
    hasEmailIdentity: Boolean,
    userTimezone: String,
    onConfirm: (LocalDate, LocalTime, String, List<String>) -> Unit,
) {
    // "Today" must be read in the account's configured timezone, not the
    // device's, since the backend interprets the picked date in that zone.
    val zone = remember(userTimezone) {
        runCatching { ZoneId.of(userTimezone) }.getOrDefault(ZoneId.systemDefault())
    }
    var form by remember {
        mutableStateOf(
            ReminderForm(date = LocalDate.now(zone), time = LocalTime.of(9, 0), recurrence = RecurrenceType.DAILY),
        )
    }
    var sendDiscordDm by rememberSaveable { mutableStateOf(hasDiscordIdentity) }
    var sendEmail by rememberSaveable { mutableStateOf(!hasDiscordIdentity && hasEmailIdentity) }

    Column(Modifier.padding(horizontal = 16.dp)) {
        WhenStep(
            form = form,
            onDateSelected = { form = form.copy(date = it) },
            onTimeSelected = { form = form.copy(time = it) },
            onRecurrenceSelected = { form = form.copy(recurrence = it) },
        )
        Spacer(Modifier.height(16.dp))
        Text(
            stringResource(R.string.dfm_destinations),
            style = MaterialTheme.typography.labelLarge,
            color = ForegroundMuted,
        )
        Spacer(Modifier.height(8.dp))
        if (hasDiscordIdentity) {
            DestinationToggleRow(
                label = stringResource(R.string.dest_discord_dm),
                checked = sendDiscordDm,
                onCheckedChange = { sendDiscordDm = it },
            )
        }
        if (hasEmailIdentity) {
            DestinationToggleRow(
                label = stringResource(R.string.dest_email),
                checked = sendEmail,
                onCheckedChange = { sendEmail = it },
            )
        }
        Spacer(Modifier.height(16.dp))
        ChronosButton(
            text = stringResource(R.string.dfm_set_reminder_confirm),
            onClick = {
                val destinations = buildList {
                    if (sendDiscordDm) add(Destination.TYPE_DISCORD_DM)
                    if (sendEmail) add(Destination.TYPE_EMAIL)
                }
                onConfirm(form.date ?: LocalDate.now(zone), form.time ?: LocalTime.of(9, 0), form.recurrence.apiString, destinations)
            },
            modifier = Modifier.fillMaxWidth(),
            enabled = (sendDiscordDm || sendEmail) && form.date != null && form.time != null,
        )
        Spacer(Modifier.height(32.dp))
    }
}

@Composable
private fun DestinationToggleRow(
    label: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, style = MaterialTheme.typography.bodyLarge, modifier = Modifier.weight(1f))
        Switch(
            checked = checked,
            onCheckedChange = onCheckedChange,
            colors = SwitchDefaults.colors(checkedTrackColor = AccentOrange),
        )
    }
}
