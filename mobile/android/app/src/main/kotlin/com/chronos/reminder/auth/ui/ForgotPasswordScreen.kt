package com.chronos.reminder.auth.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.chronos.reminder.R
import com.chronos.reminder.core.ui.components.ChronosButton
import com.chronos.reminder.core.ui.components.ChronosTextField
import com.chronos.reminder.core.ui.components.ChronosTopBar
import com.chronos.reminder.core.ui.components.ErrorBanner
import com.chronos.reminder.core.ui.theme.BackgroundMain
import com.chronos.reminder.core.ui.theme.ForegroundMuted
import com.chronos.reminder.core.ui.theme.LinkedGreen

@Composable
fun ForgotPasswordScreen(
    uiState: AuthUiState,
    onSubmit: (email: String) -> Unit,
    onBack: () -> Unit,
    onClearError: () -> Unit,
) {
    var email by rememberSaveable { mutableStateOf("") }

    Scaffold(
        containerColor = BackgroundMain,
        topBar = { ChronosTopBar(title = stringResource(R.string.reset_password_title), onBack = onBack) },
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
                    .imePadding()
                    .padding(horizontal = 16.dp),
            ) {
                Spacer(Modifier.height(16.dp))
                Text(
                    text = stringResource(R.string.reset_password_subtitle),
                    style = MaterialTheme.typography.bodyMedium,
                    color = ForegroundMuted,
                )
                Spacer(Modifier.height(20.dp))
                ChronosTextField(
                    value = email,
                    onValueChange = { email = it },
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = stringResource(R.string.email),
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
                )
                Spacer(Modifier.height(20.dp))
                if (uiState.resetEmailSent) {
                    Text(
                        text = stringResource(R.string.reset_link_sent),
                        style = MaterialTheme.typography.bodyLarge,
                        color = LinkedGreen,
                    )
                } else {
                    ChronosButton(
                        text = stringResource(R.string.send_reset_link),
                        onClick = { onSubmit(email) },
                        modifier = Modifier.fillMaxWidth(),
                        loading = uiState.loading,
                        enabled = email.contains('@'),
                    )
                }

                Spacer(Modifier.height(24.dp))
                Text(
                    text = stringResource(R.string.reset_password_discord_hint),
                    style = MaterialTheme.typography.bodySmall,
                    color = ForegroundMuted,
                )
            }

            ErrorBanner(
                message = uiState.error,
                modifier = Modifier.align(Alignment.BottomCenter),
                onDismiss = onClearError,
            )
        }
    }
}
