package com.chronos.reminder.planner.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Checklist
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.DeleteSweep
import androidx.compose.material.icons.filled.DragHandle
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Link
import androidx.compose.material.icons.filled.LightMode
import androidx.compose.material.icons.filled.DarkMode
import androidx.compose.material.icons.filled.SwapHoriz
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CheckboxDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.chronos.reminder.R
import com.chronos.reminder.core.ui.components.ChronosCard
import com.chronos.reminder.core.ui.components.ChronosTextField
import com.chronos.reminder.core.ui.components.ChronosTopBar
import com.chronos.reminder.core.ui.components.ConfirmDeleteDialog
import com.chronos.reminder.core.ui.components.ErrorBanner
import com.chronos.reminder.core.ui.theme.AccentOrange
import com.chronos.reminder.core.ui.theme.BackgroundMain
import com.chronos.reminder.core.ui.theme.ForegroundMuted
import com.chronos.reminder.dfm.data.DfmItem
import com.chronos.reminder.planner.data.PlannerItem
import com.chronos.reminder.planner.data.PlannerPeriod
import kotlinx.coroutines.launch
import sh.calvin.reorderable.ReorderableItem
import sh.calvin.reorderable.rememberReorderableLazyListState

private enum class PlannerViewMode { EDIT, CONSULT }

@Composable
fun PlannerScreen(viewModel: PlannerViewModel = hiltViewModel()) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val items by viewModel.items.collectAsStateWithLifecycle()
    val dfmItems by viewModel.dfmItems.collectAsStateWithLifecycle()
    var showClearConfirm by rememberSaveable { mutableStateOf(false) }
    var viewMode by rememberSaveable { mutableStateOf(PlannerViewMode.EDIT) }

    Scaffold(
        containerColor = BackgroundMain,
        topBar = { ChronosTopBar(title = stringResource(R.string.day_planner_title)) },
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .imePadding(),
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = stringResource(R.string.day_planner_subtitle),
                        style = MaterialTheme.typography.bodyMedium,
                        color = ForegroundMuted,
                        modifier = Modifier.weight(1f),
                    )
                }

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    val switchToConsult = viewMode == PlannerViewMode.EDIT
                    Row(
                        modifier = Modifier
                            .clip(RoundedCornerShape(20.dp))
                            .background(AccentOrange.copy(alpha = 0.12f))
                            .clickable {
                                viewMode = if (switchToConsult) PlannerViewMode.CONSULT else PlannerViewMode.EDIT
                            }
                            .padding(horizontal = 12.dp, vertical = 6.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            if (switchToConsult) Icons.Default.Checklist else Icons.Default.Edit,
                            contentDescription = stringResource(
                                if (switchToConsult) R.string.day_planner_switch_to_consult else R.string.day_planner_switch_to_edit,
                            ),
                            tint = AccentOrange,
                            modifier = Modifier.width(16.dp),
                        )
                        Spacer(Modifier.width(4.dp))
                        Text(
                            stringResource(if (switchToConsult) R.string.day_planner_consult_mode else R.string.day_planner_edit_mode),
                            style = MaterialTheme.typography.labelSmall,
                            color = AccentOrange,
                        )
                    }

                    Spacer(Modifier.weight(1f))

                    if (items.isNotEmpty()) {
                        Row(
                            modifier = Modifier
                                .clip(RoundedCornerShape(20.dp))
                                .background(MaterialTheme.colorScheme.error.copy(alpha = 0.1f))
                                .clickable { showClearConfirm = true }
                                .padding(horizontal = 12.dp, vertical = 6.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Icon(
                                Icons.Default.DeleteSweep,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.error,
                                modifier = Modifier.width(16.dp),
                            )
                            Spacer(Modifier.width(4.dp))
                            Text(
                                stringResource(R.string.day_planner_clear_all),
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.error,
                            )
                        }
                    }
                }

                if (viewMode == PlannerViewMode.EDIT) {
                    PlannerPeriodSection(
                        period = PlannerPeriod.MORNING,
                        icon = Icons.Default.LightMode,
                        label = stringResource(R.string.day_planner_morning),
                        items = items.filter { it.period == PlannerPeriod.MORNING }.sortedBy { it.position },
                        dfmItems = dfmItems,
                        onAdd = viewModel::addItem,
                        onToggle = viewModel::toggleChecked,
                        onDelete = viewModel::deleteItem,
                        onMove = { item -> viewModel.movePeriod(item, PlannerPeriod.AFTERNOON) },
                        onReordered = { ids -> viewModel.reorderWithinPeriod(PlannerPeriod.MORNING, ids) },
                    )

                    PlannerPeriodSection(
                        period = PlannerPeriod.AFTERNOON,
                        icon = Icons.Default.DarkMode,
                        label = stringResource(R.string.day_planner_afternoon),
                        items = items.filter { it.period == PlannerPeriod.AFTERNOON }.sortedBy { it.position },
                        dfmItems = dfmItems,
                        onAdd = viewModel::addItem,
                        onToggle = viewModel::toggleChecked,
                        onDelete = viewModel::deleteItem,
                        onMove = { item -> viewModel.movePeriod(item, PlannerPeriod.MORNING) },
                        onReordered = { ids -> viewModel.reorderWithinPeriod(PlannerPeriod.AFTERNOON, ids) },
                    )
                } else {
                    PlannerConsultSection(
                        morningItems = items.filter { it.period == PlannerPeriod.MORNING }.sortedBy { it.position },
                        afternoonItems = items.filter { it.period == PlannerPeriod.AFTERNOON }.sortedBy { it.position },
                        onToggle = viewModel::toggleChecked,
                        onMove = viewModel::movePeriod,
                        onReordered = { period, ids -> viewModel.reorderWithinPeriod(period, ids) },
                    )
                }

                Spacer(Modifier.height(24.dp))
            }

            ErrorBanner(
                message = state.error,
                modifier = Modifier.align(Alignment.BottomCenter),
                onDismiss = viewModel::clearError,
            )
        }
    }

    if (showClearConfirm) {
        ConfirmDeleteDialog(
            title = stringResource(R.string.day_planner_clear_all_title),
            text = stringResource(R.string.day_planner_clear_all_description),
            onConfirm = {
                viewModel.clearAll()
                showClearConfirm = false
            },
            onDismiss = { showClearConfirm = false },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun PlannerPeriodSection(
    period: PlannerPeriod,
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    label: String,
    items: List<PlannerItem>,
    dfmItems: List<DfmItem>,
    onAdd: (String, PlannerPeriod, String?) -> Unit,
    onToggle: (PlannerItem) -> Unit,
    onDelete: (PlannerItem) -> Unit,
    onMove: (PlannerItem) -> Unit,
    onReordered: (List<String>) -> Unit,
) {
    var newItemText by rememberSaveable(period) { mutableStateOf("") }
    var selectedDfmId by rememberSaveable(period) { mutableStateOf<String?>(null) }
    var showSuggestions by rememberSaveable(period) { mutableStateOf(false) }

    val localItems = remember(period) { mutableStateListOf<PlannerItem>() }
    LaunchedEffect(items) {
        localItems.clear()
        localItems.addAll(items)
    }

    val lazyListState = rememberLazyListState()
    val scope = rememberCoroutineScope()
    val reorderableState = rememberReorderableLazyListState(lazyListState) { from, to ->
        localItems.add(to.index, localItems.removeAt(from.index))
    }

    val suggestions = remember(newItemText, dfmItems) {
        val query = newItemText.trim().lowercase()
        if (query.isEmpty()) emptyList() else dfmItems.filter { it.content.lowercase().contains(query) }.take(5)
    }

    ChronosCard(modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(icon, contentDescription = null, tint = AccentOrange, modifier = Modifier.width(20.dp))
                Spacer(Modifier.width(8.dp))
                Text(label, style = MaterialTheme.typography.titleMedium)
                Spacer(Modifier.width(8.dp))
                Text(
                    "(${items.size})",
                    style = MaterialTheme.typography.bodySmall,
                    color = ForegroundMuted,
                )
            }
            Spacer(Modifier.height(12.dp))

            Box {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    ChronosTextField(
                        value = newItemText,
                        onValueChange = {
                            newItemText = it
                            selectedDfmId = null
                            showSuggestions = true
                        },
                        modifier = Modifier.weight(1f),
                        placeholder = stringResource(R.string.day_planner_add_placeholder),
                        keyboardOptions = KeyboardOptions(imeAction = androidx.compose.ui.text.input.ImeAction.Done),
                        keyboardActions = KeyboardActions(onDone = {
                            if (newItemText.isNotBlank()) {
                                onAdd(newItemText.trim(), period, selectedDfmId)
                                newItemText = ""
                                selectedDfmId = null
                                showSuggestions = false
                            }
                        }),
                    )
                    Spacer(Modifier.width(8.dp))
                    IconButton(
                        onClick = {
                            if (newItemText.isNotBlank()) {
                                onAdd(newItemText.trim(), period, selectedDfmId)
                                newItemText = ""
                                selectedDfmId = null
                                showSuggestions = false
                            }
                        },
                        enabled = newItemText.isNotBlank(),
                    ) {
                        Icon(Icons.Default.Add, contentDescription = stringResource(R.string.day_planner_add_item), tint = AccentOrange)
                    }
                }

                if (showSuggestions && suggestions.isNotEmpty()) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(top = 56.dp),
                    ) {
                        ChronosCard(modifier = Modifier.fillMaxWidth()) {
                            Column {
                                suggestions.forEach { suggestion ->
                                    Row(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .clickable {
                                                newItemText = suggestion.content
                                                selectedDfmId = suggestion.id
                                                showSuggestions = false
                                            }
                                            .padding(horizontal = 12.dp, vertical = 10.dp),
                                        verticalAlignment = Alignment.CenterVertically,
                                    ) {
                                        Icon(
                                            Icons.Default.Link,
                                            contentDescription = null,
                                            tint = AccentOrange,
                                            modifier = Modifier.width(16.dp),
                                        )
                                        Spacer(Modifier.width(8.dp))
                                        Text(
                                            suggestion.content,
                                            style = MaterialTheme.typography.bodyMedium,
                                            modifier = Modifier.weight(1f),
                                        )
                                    }
                                }
                            }
                        }
                    }
                }
            }

            if (selectedDfmId != null) {
                Spacer(Modifier.height(4.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Default.Link, contentDescription = null, tint = AccentOrange, modifier = Modifier.width(14.dp))
                    Spacer(Modifier.width(4.dp))
                    Text(
                        stringResource(R.string.day_planner_will_link),
                        style = MaterialTheme.typography.labelSmall,
                        color = AccentOrange,
                    )
                }
            }

            Spacer(Modifier.height(12.dp))

            if (localItems.isEmpty()) {
                Text(
                    stringResource(R.string.day_planner_empty_column),
                    style = MaterialTheme.typography.bodySmall,
                    color = ForegroundMuted,
                    modifier = Modifier.padding(vertical = 8.dp),
                )
            } else {
                LazyColumn(
                    state = lazyListState,
                    modifier = Modifier.heightIn(max = 400.dp),
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    items(localItems, key = { it.id }) { item ->
                        ReorderableItem(reorderableState, key = item.id) { _ ->
                            PlannerRow(
                                item = item,
                                onToggle = { onToggle(item) },
                                onDelete = { onDelete(item) },
                                onMove = { onMove(item) },
                                dragHandleModifier = Modifier.draggableHandle(
                                    onDragStopped = {
                                        scope.launch { onReordered(localItems.map { it.id }) }
                                    },
                                ),
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun PlannerRow(
    item: PlannerItem,
    onToggle: () -> Unit,
    onDelete: () -> Unit,
    onMove: () -> Unit,
    dragHandleModifier: Modifier,
) {
    val haptic = LocalHapticFeedback.current
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            Icons.Default.DragHandle,
            contentDescription = stringResource(R.string.day_planner_drag_handle),
            tint = ForegroundMuted,
            modifier = dragHandleModifier.width(20.dp),
        )
        Checkbox(
            checked = item.checked,
            onCheckedChange = {
                haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                onToggle()
            },
            colors = CheckboxDefaults.colors(checkedColor = AccentOrange),
        )
        if (item.dfmItemId != null) {
            Icon(
                Icons.Default.Link,
                contentDescription = stringResource(R.string.day_planner_linked_to_dfm),
                tint = AccentOrange,
                modifier = Modifier.width(14.dp),
            )
            Spacer(Modifier.width(4.dp))
        }
        Text(
            text = item.content,
            style = MaterialTheme.typography.bodyMedium.copy(
                textDecoration = if (item.checked) TextDecoration.LineThrough else TextDecoration.None,
            ),
            color = if (item.checked) ForegroundMuted else MaterialTheme.colorScheme.onBackground,
            modifier = Modifier.weight(1f).padding(start = 4.dp),
        )
        IconButton(onClick = onMove) {
            Icon(
                Icons.Default.SwapHoriz,
                contentDescription = stringResource(R.string.day_planner_move_period),
                tint = ForegroundMuted,
            )
        }
        IconButton(onClick = onDelete) {
            Icon(
                Icons.Default.Close,
                contentDescription = stringResource(R.string.day_planner_delete_item),
                tint = ForegroundMuted,
            )
        }
    }
}

@Composable
private fun PlannerConsultSection(
    morningItems: List<PlannerItem>,
    afternoonItems: List<PlannerItem>,
    onToggle: (PlannerItem) -> Unit,
    onMove: (PlannerItem, PlannerPeriod) -> Unit,
    onReordered: (PlannerPeriod, List<String>) -> Unit,
) {
    ChronosCard(modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp)) {
            PlannerConsultPeriodBlock(
                period = PlannerPeriod.MORNING,
                icon = Icons.Default.LightMode,
                label = stringResource(R.string.day_planner_morning),
                items = morningItems,
                onToggle = onToggle,
                onMove = { item -> onMove(item, PlannerPeriod.AFTERNOON) },
                onReordered = { ids -> onReordered(PlannerPeriod.MORNING, ids) },
            )
            Spacer(Modifier.height(16.dp))
            PlannerConsultPeriodBlock(
                period = PlannerPeriod.AFTERNOON,
                icon = Icons.Default.DarkMode,
                label = stringResource(R.string.day_planner_afternoon),
                items = afternoonItems,
                onToggle = onToggle,
                onMove = { item -> onMove(item, PlannerPeriod.MORNING) },
                onReordered = { ids -> onReordered(PlannerPeriod.AFTERNOON, ids) },
            )
        }
    }
}

@Composable
private fun PlannerConsultPeriodBlock(
    period: PlannerPeriod,
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    label: String,
    items: List<PlannerItem>,
    onToggle: (PlannerItem) -> Unit,
    onMove: (PlannerItem) -> Unit,
    onReordered: (List<String>) -> Unit,
) {
    val localItems = remember(period) { mutableStateListOf<PlannerItem>() }
    LaunchedEffect(items) {
        localItems.clear()
        localItems.addAll(items)
    }

    val lazyListState = rememberLazyListState()
    val scope = rememberCoroutineScope()
    val reorderableState = rememberReorderableLazyListState(lazyListState) { from, to ->
        localItems.add(to.index, localItems.removeAt(from.index))
    }

    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(icon, contentDescription = null, tint = AccentOrange, modifier = Modifier.width(20.dp))
        Spacer(Modifier.width(8.dp))
        Text(label, style = MaterialTheme.typography.titleMedium)
    }
    Spacer(Modifier.height(4.dp))

    if (localItems.isEmpty()) {
        Text(
            stringResource(R.string.day_planner_empty_column),
            style = MaterialTheme.typography.bodySmall,
            color = ForegroundMuted,
            modifier = Modifier.padding(vertical = 4.dp),
        )
    } else {
        LazyColumn(
            state = lazyListState,
            modifier = Modifier.heightIn(max = 400.dp),
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            items(localItems, key = { it.id }) { item ->
                ReorderableItem(reorderableState, key = item.id) { _ ->
                    PlannerConsultRow(
                        item = item,
                        onToggle = { onToggle(item) },
                        onMove = { onMove(item) },
                        dragHandleModifier = Modifier.draggableHandle(
                            onDragStopped = {
                                scope.launch { onReordered(localItems.map { it.id }) }
                            },
                        ),
                    )
                }
            }
        }
    }
}

@Composable
private fun PlannerConsultRow(
    item: PlannerItem,
    onToggle: () -> Unit,
    onMove: () -> Unit,
    dragHandleModifier: Modifier,
) {
    val haptic = LocalHapticFeedback.current
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            Icons.Default.DragHandle,
            contentDescription = stringResource(R.string.day_planner_drag_handle),
            tint = ForegroundMuted,
            modifier = dragHandleModifier.width(20.dp),
        )
        Checkbox(
            checked = item.checked,
            onCheckedChange = {
                haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                onToggle()
            },
            colors = CheckboxDefaults.colors(checkedColor = AccentOrange),
        )
        if (item.dfmItemId != null) {
            Icon(
                Icons.Default.Link,
                contentDescription = stringResource(R.string.day_planner_linked_to_dfm),
                tint = AccentOrange,
                modifier = Modifier.width(14.dp),
            )
            Spacer(Modifier.width(4.dp))
        }
        Text(
            text = item.content,
            style = MaterialTheme.typography.bodyMedium.copy(
                textDecoration = if (item.checked) TextDecoration.LineThrough else TextDecoration.None,
            ),
            color = if (item.checked) ForegroundMuted else MaterialTheme.colorScheme.onBackground,
            modifier = Modifier.weight(1f).padding(start = 4.dp),
        )
        IconButton(onClick = onMove) {
            Icon(
                Icons.Default.SwapHoriz,
                contentDescription = stringResource(R.string.day_planner_move_period),
                tint = ForegroundMuted,
            )
        }
    }
}
