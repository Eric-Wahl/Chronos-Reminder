package com.chronos.reminder.account.ui

import android.app.Activity
import android.app.LocaleManager
import android.content.Context
import android.net.Uri
import android.os.Build
import androidx.browser.customtabs.CustomTabsIntent
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.RadioButton
import androidx.compose.material3.RadioButtonDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.platform.LocalContext
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil3.compose.AsyncImage
import com.chronos.reminder.BuildConfig
import com.chronos.reminder.R
import com.chronos.reminder.auth.ui.TimezonePickerSheetContent
import com.chronos.reminder.auth.ui.discordOAuthUrl
import com.chronos.reminder.core.ui.components.ChronosButton
import com.chronos.reminder.core.ui.components.ChronosButtonStyle
import com.chronos.reminder.core.ui.components.ChronosCard
import com.chronos.reminder.core.ui.components.ChronosTextField
import com.chronos.reminder.core.ui.components.ChronosTopBar
import com.chronos.reminder.core.ui.components.ConfirmDeleteDialog
import com.chronos.reminder.core.ui.components.ErrorBanner
import com.chronos.reminder.core.ui.theme.AccentOrange
import com.chronos.reminder.core.ui.theme.BackgroundCard
import com.chronos.reminder.core.ui.theme.BackgroundMain
import com.chronos.reminder.core.ui.theme.ForegroundMuted
import com.chronos.reminder.core.ui.theme.LinkedGreen

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AccountScreen(
    onOpenApiKeys: () -> Unit,
    onOpenLinks: () -> Unit,
    onOpenAbout: () -> Unit,
    onLogout: () -> Unit,
    onAccountDeleted: () -> Unit,
    pendingDiscordLinkCode: String? = null,
    onPendingDiscordLinkConsumed: () -> Unit = {},
    viewModel: AccountViewModel = hiltViewModel(),
) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val account by viewModel.account.collectAsStateWithLifecycle()
    val context = LocalContext.current

    var showTimezoneSheet by rememberSaveable { mutableStateOf(false) }
    var showLanguageDialog by rememberSaveable { mutableStateOf(false) }
    var showDeleteDialog by rememberSaveable { mutableStateOf(false) }
    var deleteConfirmText by rememberSaveable { mutableStateOf("") }
    var newUsername by rememberSaveable { mutableStateOf("") }
    var newEmail by rememberSaveable { mutableStateOf("") }
    var showPasswordSection by rememberSaveable { mutableStateOf(false) }
    var oldPassword by rememberSaveable { mutableStateOf("") }
    var newPassword by rememberSaveable { mutableStateOf("") }
    var confirmNewPassword by rememberSaveable { mutableStateOf("") }
    var addAppEmail by rememberSaveable { mutableStateOf("") }
    var addAppPassword by rememberSaveable { mutableStateOf("") }

    val snackbarHostState = remember { SnackbarHostState() }
    val tzUpdated = stringResource(R.string.timezone_updated)
    val usernameUpdated = stringResource(R.string.username_updated)
    val emailUpdated = stringResource(R.string.email_updated)
    val passwordChanged = stringResource(R.string.password_changed)
    val appLoginAdded = stringResource(R.string.app_login_added)
    val discordImagePrefUpdated = stringResource(R.string.discord_image_pref_updated)
    val discordSnoozePrefUpdated = stringResource(R.string.discord_snooze_pref_updated)

    val hasAppCredentials = account?.email != null
    val discordIdentity = account?.identities?.firstOrNull { it.provider == "discord" }
    val mobileIdentity = account?.identities?.firstOrNull { it.provider == "mobile" }

    LaunchedEffect(state.successMessage) {
        state.successMessage?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.clearMessages()
        }
    }
    LaunchedEffect(state.accountDeleted) {
        if (state.accountDeleted) onAccountDeleted()
    }
    val discordLinkedMsg = stringResource(R.string.discord_linked)
    LaunchedEffect(pendingDiscordLinkCode) {
        pendingDiscordLinkCode?.let {
            viewModel.linkDiscord(it, discordLinkedMsg)
            onPendingDiscordLinkConsumed()
        }
    }

    Scaffold(
        containerColor = BackgroundMain,
        topBar = { ChronosTopBar(title = stringResource(R.string.account_title)) },
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
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 16.dp),
            ) {
                // --- Profile ---
                SectionHeader(stringResource(R.string.section_profile))
                ChronosCard(modifier = Modifier.fillMaxWidth()) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(16.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        var avatarLoadFailed by remember { mutableStateOf(false) }
                        val avatarUrl = if (!discordIdentity?.avatar.isNullOrBlank() && !discordIdentity?.externalId.isNullOrBlank()) {
                            "https://cdn.discordapp.com/avatars/${discordIdentity.externalId}/${discordIdentity.avatar}.png"
                        } else null

                        if (avatarUrl != null && !avatarLoadFailed) {
                            AsyncImage(
                                model = avatarUrl,
                                contentDescription = stringResource(R.string.avatar),
                                modifier = Modifier
                                    .size(48.dp)
                                    .clip(CircleShape),
                                onError = { avatarLoadFailed = true },
                            )
                        } else {
                            Icon(
                                Icons.Default.Person,
                                contentDescription = stringResource(R.string.avatar),
                                modifier = Modifier.size(48.dp),
                                tint = ForegroundMuted,
                            )
                        }
                        Spacer(Modifier.width(12.dp))
                        Column {
                            Text(
                                text = account?.username ?: discordIdentity?.username ?: "—",
                                style = MaterialTheme.typography.titleMedium,
                            )
                            account?.email?.let { email ->
                                Text(email, style = MaterialTheme.typography.bodyMedium, color = ForegroundMuted)
                            }
                            discordIdentity?.username?.let { tag ->
                                Text(tag, style = MaterialTheme.typography.bodyMedium, color = ForegroundMuted)
                            }
                        }
                    }
                }

                // --- Email verification banner ---
                if (account?.email != null && account?.emailVerified == false) {
                    val resendSentMsg = stringResource(R.string.resend_verification_sent)
                    Spacer(Modifier.height(8.dp))
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(bottom = 4.dp)
                            .background(
                                color = AccentOrange.copy(alpha = 0.12f),
                                shape = RoundedCornerShape(12.dp),
                            )
                            .padding(12.dp),
                    ) {
                        Column {
                            Text(
                                text = stringResource(R.string.email_not_verified),
                                style = MaterialTheme.typography.titleSmall,
                                color = AccentOrange,
                            )
                            Spacer(Modifier.height(2.dp))
                            Text(
                                text = stringResource(R.string.email_not_verified_desc),
                                style = MaterialTheme.typography.bodySmall,
                                color = ForegroundMuted,
                            )
                            Spacer(Modifier.height(8.dp))
                            ChronosButton(
                                text = stringResource(R.string.resend_verification),
                                onClick = { viewModel.resendVerificationEmail(account!!.email!!, resendSentMsg) },
                                style = ChronosButtonStyle.Secondary,
                            )
                        }
                    }
                }

                // --- Preferences (timezone + Discord subcategory) ---
                SectionHeader(stringResource(R.string.section_preferences))
                ChronosCard(modifier = Modifier.fillMaxWidth()) {
                    Column(Modifier.padding(16.dp)) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    viewModel.loadTimezones()
                                    showTimezoneSheet = true
                                },
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(
                                text = stringResource(
                                    R.string.current_timezone,
                                    account?.timezone?.ianaLocation ?: "—",
                                ),
                                style = MaterialTheme.typography.bodyLarge,
                                modifier = Modifier.weight(1f),
                            )
                            Icon(
                                Icons.AutoMirrored.Filled.KeyboardArrowRight,
                                contentDescription = null,
                                tint = ForegroundMuted,
                            )
                        }

                        if (discordIdentity != null) {
                            HorizontalDivider(modifier = Modifier.padding(vertical = 16.dp))
                            Text(
                                stringResource(R.string.section_discord_preferences),
                                style = MaterialTheme.typography.labelLarge,
                                color = ForegroundMuted,
                            )
                            Spacer(Modifier.height(8.dp))

                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Column(Modifier.weight(1f)) {
                                    Text(
                                        stringResource(R.string.discord_send_image),
                                        style = MaterialTheme.typography.bodyLarge,
                                    )
                                    Text(
                                        stringResource(R.string.discord_send_image_desc),
                                        style = MaterialTheme.typography.bodySmall,
                                        color = ForegroundMuted,
                                    )
                                }
                                Switch(
                                    checked = account?.preferences?.discordSendImageOrDefault ?: true,
                                    onCheckedChange = {
                                        viewModel.updateDiscordSendImagePreference(it, discordImagePrefUpdated)
                                    },
                                    colors = SwitchDefaults.colors(checkedTrackColor = AccentOrange),
                                )
                            }

                            Spacer(Modifier.height(16.dp))

                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Column(Modifier.weight(1f)) {
                                    Text(
                                        stringResource(R.string.discord_enable_snooze),
                                        style = MaterialTheme.typography.bodyLarge,
                                    )
                                    Text(
                                        stringResource(R.string.discord_enable_snooze_desc),
                                        style = MaterialTheme.typography.bodySmall,
                                        color = ForegroundMuted,
                                    )
                                }
                                Switch(
                                    checked = account?.preferences?.discordEnableSnoozeOrDefault ?: true,
                                    onCheckedChange = {
                                        viewModel.updateDiscordEnableSnoozePreference(it, discordSnoozePrefUpdated)
                                    },
                                    colors = SwitchDefaults.colors(checkedTrackColor = AccentOrange),
                                )
                            }
                        }
                    }
                }

                // --- Username (available to all users) ---
                SectionHeader(stringResource(R.string.section_username))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    ChronosTextField(
                        value = newUsername,
                        onValueChange = { newUsername = it },
                        modifier = Modifier.weight(1f),
                        placeholder = account?.username ?: discordIdentity?.username ?: stringResource(R.string.username),
                    )
                    Spacer(Modifier.width(8.dp))
                    ChronosButton(
                        text = stringResource(R.string.save),
                        onClick = {
                            viewModel.updateUsername(newUsername, usernameUpdated)
                            newUsername = ""
                        },
                        enabled = newUsername.isNotBlank(),
                    )
                }

                if (hasAppCredentials) {
                    // --- Email ---
                    SectionHeader(stringResource(R.string.section_email))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        ChronosTextField(
                            value = newEmail,
                            onValueChange = { newEmail = it },
                            modifier = Modifier.weight(1f),
                            placeholder = account?.email ?: stringResource(R.string.email),
                        )
                        Spacer(Modifier.width(8.dp))
                        ChronosButton(
                            text = stringResource(R.string.save),
                            onClick = {
                                viewModel.updateEmail(newEmail, emailUpdated)
                                newEmail = ""
                            },
                            enabled = newEmail.contains('@'),
                        )
                    }

                    // --- Password ---
                    SectionHeader(
                        title = stringResource(R.string.section_password),
                        modifier = Modifier.clickable { showPasswordSection = !showPasswordSection },
                    )
                    if (showPasswordSection) {
                        ChronosTextField(
                            value = oldPassword,
                            onValueChange = { oldPassword = it },
                            modifier = Modifier.fillMaxWidth(),
                            placeholder = stringResource(R.string.old_password),
                            visualTransformation = PasswordVisualTransformation(),
                        )
                        Spacer(Modifier.height(8.dp))
                        ChronosTextField(
                            value = newPassword,
                            onValueChange = { newPassword = it },
                            modifier = Modifier.fillMaxWidth(),
                            placeholder = stringResource(R.string.new_password),
                            visualTransformation = PasswordVisualTransformation(),
                        )
                        Spacer(Modifier.height(8.dp))
                        ChronosTextField(
                            value = confirmNewPassword,
                            onValueChange = { confirmNewPassword = it },
                            modifier = Modifier.fillMaxWidth(),
                            placeholder = stringResource(R.string.confirm_new_password),
                            visualTransformation = PasswordVisualTransformation(),
                            isError = confirmNewPassword.isNotEmpty() && confirmNewPassword != newPassword,
                        )
                        Spacer(Modifier.height(8.dp))
                        ChronosButton(
                            text = stringResource(R.string.save),
                            onClick = {
                                viewModel.changePassword(oldPassword, newPassword, passwordChanged)
                                oldPassword = ""
                                newPassword = ""
                                confirmNewPassword = ""
                            },
                            modifier = Modifier.fillMaxWidth(),
                            enabled = oldPassword.isNotBlank() &&
                                newPassword.isNotBlank() &&
                                newPassword == confirmNewPassword,
                        )
                    }
                } else {
                    // No email/password identity yet (e.g. Discord-first account).
                    // Let the user add one so the same account works on web & mobile.
                    SectionHeader(stringResource(R.string.section_add_login))
                    Text(
                        text = stringResource(R.string.add_login_help),
                        style = MaterialTheme.typography.bodySmall,
                        color = ForegroundMuted,
                        modifier = Modifier.padding(bottom = 8.dp),
                    )
                    ChronosTextField(
                        value = addAppEmail,
                        onValueChange = { addAppEmail = it },
                        modifier = Modifier.fillMaxWidth(),
                        placeholder = stringResource(R.string.email),
                    )
                    Spacer(Modifier.height(8.dp))
                    ChronosTextField(
                        value = addAppPassword,
                        onValueChange = { addAppPassword = it },
                        modifier = Modifier.fillMaxWidth(),
                        placeholder = stringResource(R.string.password),
                        visualTransformation = PasswordVisualTransformation(),
                        isError = addAppPassword.isNotEmpty() && addAppPassword.length < 8,
                    )
                    Spacer(Modifier.height(8.dp))
                    ChronosButton(
                        text = stringResource(R.string.add_login_submit),
                        onClick = {
                            val username = account?.username ?: discordIdentity?.username ?: ""
                            viewModel.addAppIdentity(addAppEmail, username, addAppPassword, appLoginAdded)
                            addAppEmail = ""
                            addAppPassword = ""
                        },
                        modifier = Modifier.fillMaxWidth(),
                        enabled = addAppEmail.contains('@') &&
                            addAppPassword.length >= 8,
                    )
                }

                // --- Identities ---
                SectionHeader(stringResource(R.string.section_identities))
                ChronosCard(modifier = Modifier.fillMaxWidth()) {
                    Column(Modifier.padding(16.dp)) {
                        IdentityRow(
                            label = stringResource(R.string.provider_app),
                            linked = hasAppCredentials,
                        )
                        Spacer(Modifier.height(8.dp))
                        IdentityRow(
                            label = stringResource(R.string.provider_discord),
                            linked = discordIdentity != null,
                            detail = discordIdentity?.username,
                            onConnect = {
                                if (BuildConfig.DISCORD_CLIENT_ID.isNotBlank()) {
                                    CustomTabsIntent.Builder().build()
                                        .launchUrl(context, Uri.parse(discordOAuthUrl()))
                                }
                            },
                        )
                        if (mobileIdentity != null) {
                            Spacer(Modifier.height(8.dp))
                            IdentityRow(
                                label = stringResource(R.string.provider_mobile),
                                linked = true,
                                detail = mobileIdentity.username,
                            )
                        }
                    }
                }

                // --- API Keys ---
                SectionHeader(stringResource(R.string.section_api_keys))
                ChronosCard(modifier = Modifier.fillMaxWidth(), onClick = onOpenApiKeys) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(16.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            stringResource(R.string.manage_api_keys),
                            style = MaterialTheme.typography.bodyLarge,
                            modifier = Modifier.weight(1f),
                        )
                        Icon(
                            Icons.AutoMirrored.Filled.KeyboardArrowRight,
                            contentDescription = null,
                            tint = ForegroundMuted,
                        )
                    }
                }

                // --- Language ---
                SectionHeader(stringResource(R.string.section_language))
                ChronosCard(modifier = Modifier.fillMaxWidth(), onClick = { showLanguageDialog = true }) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(16.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            when (currentLocaleTag(context)) {
                                "fr" -> stringResource(R.string.lang_fr)
                                "es" -> stringResource(R.string.lang_es)
                                else -> stringResource(R.string.lang_en)
                            },
                            style = MaterialTheme.typography.bodyLarge,
                            modifier = Modifier.weight(1f),
                        )
                        Icon(
                            Icons.AutoMirrored.Filled.KeyboardArrowRight,
                            contentDescription = null,
                            tint = ForegroundMuted,
                        )
                    }
                }

                // --- About ---
                SectionHeader(stringResource(R.string.section_about))
                ChronosCard(modifier = Modifier.fillMaxWidth(), onClick = onOpenAbout) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(16.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            stringResource(R.string.about_title),
                            style = MaterialTheme.typography.bodyLarge,
                            modifier = Modifier.weight(1f),
                        )
                        Icon(
                            Icons.AutoMirrored.Filled.KeyboardArrowRight,
                            contentDescription = null,
                            tint = ForegroundMuted,
                        )
                    }
                }

                // --- Links & Resources ---
                SectionHeader(stringResource(R.string.section_links))
                ChronosCard(modifier = Modifier.fillMaxWidth(), onClick = onOpenLinks) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(16.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            stringResource(R.string.links_screen_title),
                            style = MaterialTheme.typography.bodyLarge,
                            modifier = Modifier.weight(1f),
                        )
                        Icon(
                            Icons.AutoMirrored.Filled.KeyboardArrowRight,
                            contentDescription = null,
                            tint = ForegroundMuted,
                        )
                    }
                }

                Spacer(Modifier.height(24.dp))
                ChronosButton(
                    text = stringResource(R.string.sign_out),
                    onClick = onLogout,
                    modifier = Modifier.fillMaxWidth(),
                    style = ChronosButtonStyle.Secondary,
                )

                // --- Danger zone ---
                SectionHeader(stringResource(R.string.section_danger))
                ChronosButton(
                    text = stringResource(R.string.delete_account),
                    onClick = {
                        deleteConfirmText = ""
                        showDeleteDialog = true
                    },
                    modifier = Modifier.fillMaxWidth(),
                    style = ChronosButtonStyle.Destructive,
                )
                Spacer(Modifier.height(32.dp))
            }

            ErrorBanner(
                message = state.error,
                modifier = Modifier.align(Alignment.BottomCenter),
                onDismiss = viewModel::clearMessages,
            )
        }
    }

    if (showLanguageDialog) {
        val languages = listOf(
            "en" to stringResource(R.string.lang_en),
            "fr" to stringResource(R.string.lang_fr),
            "es" to stringResource(R.string.lang_es),
        )
        val currentTag = currentLocaleTag(context)
        AlertDialog(
            onDismissRequest = { showLanguageDialog = false },
            title = { Text(stringResource(R.string.section_language)) },
            text = {
                Column {
                    languages.forEach { (tag, label) ->
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    applyLocale(context, tag)
                                    showLanguageDialog = false
                                }
                                .padding(vertical = 4.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            RadioButton(
                                selected = currentTag == tag || (tag == "en" && currentTag.isEmpty()),
                                onClick = {
                                    applyLocale(context, tag)
                                    showLanguageDialog = false
                                },
                                colors = RadioButtonDefaults.colors(selectedColor = AccentOrange),
                            )
                            Spacer(Modifier.width(8.dp))
                            Text(label, style = MaterialTheme.typography.bodyLarge)
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { showLanguageDialog = false }) {
                    Text(stringResource(R.string.cancel))
                }
            },
        )
    }

    if (showTimezoneSheet) {
        ModalBottomSheet(onDismissRequest = { showTimezoneSheet = false }, containerColor = BackgroundCard) {
            TimezonePickerSheetContent(
                timezones = state.timezones,
                onSelect = {
                    viewModel.updateTimezone(it, tzUpdated)
                    showTimezoneSheet = false
                },
            )
        }
    }

    // Merge confirmation dialog
    val mergeSuccessMsg = stringResource(R.string.merge_success)
    if (state.pendingMergeToken != null) {
        AlertDialog(
            onDismissRequest = { viewModel.dismissMerge() },
            title = { Text(stringResource(R.string.merge_title)) },
            text = {
                Text(
                    stringResource(
                        R.string.merge_description,
                        state.pendingMergeDiscordUsername ?: "",
                    )
                )
            },
            confirmButton = {
                TextButton(onClick = { viewModel.confirmMerge(mergeSuccessMsg) }) {
                    Text(stringResource(R.string.merge_confirm))
                }
            },
            dismissButton = {
                TextButton(onClick = { viewModel.dismissMerge() }) {
                    Text(stringResource(R.string.cancel))
                }
            },
        )
    }

    if (showDeleteDialog) {
        val confirmWord = stringResource(R.string.delete_confirm_word)
        ConfirmDeleteDialog(
            title = stringResource(R.string.delete_account_title),
            text = stringResource(R.string.delete_account_text),
            confirmEnabled = deleteConfirmText == confirmWord,
            onConfirm = {
                showDeleteDialog = false
                viewModel.deleteAccount()
            },
            onDismiss = { showDeleteDialog = false },
            extraContent = {
                Spacer(Modifier.height(12.dp))
                ChronosTextField(
                    value = deleteConfirmText,
                    onValueChange = { deleteConfirmText = it },
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = confirmWord,
                )
            },
        )
    }
}

private fun currentLocaleTag(context: Context): String {
    return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
        val localeManager = context.getSystemService(LocaleManager::class.java)
        val appLocale = localeManager?.applicationLocales?.get(0)?.language
        // Fall back to the actual displayed locale when no app-specific override is set
        if (!appLocale.isNullOrEmpty()) appLocale
        else context.resources.configuration.locales.get(0)?.language ?: ""
    } else {
        val prefs = context.getSharedPreferences("chronos_prefs", Context.MODE_PRIVATE)
        prefs.getString("locale_tag", null) ?: java.util.Locale.getDefault().language
    }
}

private fun applyLocale(context: Context, tag: String) {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
        val localeManager = context.getSystemService(LocaleManager::class.java)
        localeManager?.applicationLocales = android.os.LocaleList.forLanguageTags(tag)
    } else {
        val prefs = context.getSharedPreferences("chronos_prefs", Context.MODE_PRIVATE)
        prefs.edit().putString("locale_tag", tag).apply()
        (context as? Activity)?.recreate()
    }
}

@Composable
private fun SectionHeader(title: String, modifier: Modifier = Modifier) {
    Column(modifier = modifier.fillMaxWidth()) {
        Spacer(Modifier.height(20.dp))
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            color = AccentOrange,
        )
        Spacer(Modifier.height(8.dp))
    }
}

@Composable
private fun IdentityRow(
    label: String,
    linked: Boolean,
    detail: String? = null,
    onConnect: (() -> Unit)? = null,
) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text(label, style = MaterialTheme.typography.bodyLarge, modifier = Modifier.weight(1f))
        detail?.let {
            Text(it, style = MaterialTheme.typography.bodyMedium, color = ForegroundMuted)
            Spacer(Modifier.width(8.dp))
        }
        if (linked) {
            Text(
                stringResource(R.string.linked_badge),
                style = MaterialTheme.typography.labelSmall,
                color = LinkedGreen,
            )
        } else if (onConnect != null) {
            TextButton(onClick = onConnect) {
                Text(
                    stringResource(R.string.connect_discord),
                    style = MaterialTheme.typography.labelLarge,
                    color = AccentOrange,
                )
            }
        }
    }
}
