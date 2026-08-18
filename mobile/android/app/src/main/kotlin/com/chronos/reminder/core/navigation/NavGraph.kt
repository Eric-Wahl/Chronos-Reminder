package com.chronos.reminder.core.navigation

import androidx.compose.animation.AnimatedContentTransitionScope
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.animation.expandVertically
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkVertically
import androidx.compose.ui.unit.IntSize
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CalendarToday
import androidx.compose.material.icons.filled.Checklist
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.sp
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.navigation.NavDestination.Companion.hasRoute
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.navigation
import com.chronos.reminder.R
import com.chronos.reminder.account.ui.AccountScreen
import com.chronos.reminder.account.ui.ApiKeysScreen
import com.chronos.reminder.auth.ui.AuthViewModel
import com.chronos.reminder.auth.ui.DiscordSetupScreen
import com.chronos.reminder.auth.ui.ForgotPasswordScreen
import com.chronos.reminder.auth.ui.LoginScreen
import com.chronos.reminder.auth.ui.RegisterScreen
import com.chronos.reminder.core.ui.screen.AboutScreen
import com.chronos.reminder.core.ui.screen.LinksScreen
import com.chronos.reminder.core.ui.screen.PrivacyScreen
import com.chronos.reminder.core.ui.screen.TermsScreen
import com.chronos.reminder.core.ui.theme.AccentOrange
import com.chronos.reminder.core.ui.theme.BackgroundCard
import com.chronos.reminder.core.ui.theme.BackgroundMain
import com.chronos.reminder.core.ui.theme.BackgroundMuted
import com.chronos.reminder.core.ui.theme.ForegroundMain
import com.chronos.reminder.core.ui.theme.ForegroundMuted
import com.chronos.reminder.dfm.ui.DfmScreen
import com.chronos.reminder.home.ui.HomeScreen
import com.chronos.reminder.planner.ui.PlannerScreen
import com.chronos.reminder.reminders.ui.create.CreateReminderScreen
import com.chronos.reminder.reminders.ui.detail.ReminderDetailScreen
import com.chronos.reminder.reminders.ui.list.RemindersScreen

private data class BottomNavItem(
    val route: Screen,
    val icon: ImageVector,
    val labelRes: Int,
)

private val bottomNavItems = listOf(
    BottomNavItem(Screen.Home, Icons.Default.Home, R.string.nav_home),
    BottomNavItem(Screen.Reminders, Icons.Default.Schedule, R.string.nav_reminders),
    BottomNavItem(Screen.Dfm, Icons.Default.Checklist, R.string.nav_dfm),
    BottomNavItem(Screen.DayPlanner, Icons.Default.CalendarToday, R.string.nav_day_planner),
    BottomNavItem(Screen.Account, Icons.Default.Person, R.string.nav_account),
)

// Hourglass-sand animation:
// Enter  — content expands upward from the bottom, like sand collecting in the lower bowl.
// Exit   — content shrinks downward toward the bottom, like sand draining through the neck.
// The combination makes the transition look like contents pour through an hourglass neck.
private val sandSpec = tween<Float>(380, easing = FastOutSlowInEasing)
private val sandIntSpec = tween<IntSize>(380, easing = FastOutSlowInEasing)

private val sandEnter: AnimatedContentTransitionScope<*>.() -> EnterTransition = {
    expandVertically(
        animationSpec = sandIntSpec,
        expandFrom = androidx.compose.ui.Alignment.Bottom,
    ) + fadeIn(animationSpec = sandSpec)
}

private val sandExit: AnimatedContentTransitionScope<*>.() -> ExitTransition = {
    shrinkVertically(
        animationSpec = sandIntSpec,
        shrinkTowards = androidx.compose.ui.Alignment.Bottom,
    ) + fadeOut(animationSpec = sandSpec)
}

private val sandPopEnter: AnimatedContentTransitionScope<*>.() -> EnterTransition = {
    expandVertically(
        animationSpec = sandIntSpec,
        expandFrom = androidx.compose.ui.Alignment.Top,
    ) + fadeIn(animationSpec = sandSpec)
}

private val sandPopExit: AnimatedContentTransitionScope<*>.() -> ExitTransition = {
    shrinkVertically(
        animationSpec = sandIntSpec,
        shrinkTowards = androidx.compose.ui.Alignment.Top,
    ) + fadeOut(animationSpec = sandSpec)
}

@Composable
fun ChronosNavGraph(
    navController: NavHostController,
    authViewModel: AuthViewModel,
    uiState: com.chronos.reminder.auth.ui.AuthUiState,
    isOnline: Boolean,
    pendingReminderId: String?,
    onPendingReminderConsumed: () -> Unit,
    pendingDiscordLinkCode: String? = null,
    onPendingDiscordLinkConsumed: () -> Unit = {},
    onDiscordUnconfigured: () -> Unit,
) {
    var lastLoggedIn by rememberSaveable { mutableStateOf(uiState.isLoggedIn) }
    LaunchedEffect(uiState.isLoggedIn) {
        if (lastLoggedIn == uiState.isLoggedIn) return@LaunchedEffect
        lastLoggedIn = uiState.isLoggedIn
        if (uiState.isLoggedIn) {
            navController.navigate(Screen.MainGraph) {
                popUpTo(Screen.AuthGraph) { inclusive = true }
            }
        } else {
            navController.navigate(Screen.AuthGraph) {
                popUpTo(navController.graph.findStartDestination().id) { inclusive = true }
            }
        }
    }

    LaunchedEffect(pendingReminderId, uiState.isLoggedIn) {
        if (pendingReminderId != null && uiState.isLoggedIn) {
            navController.navigate(Screen.ReminderDetail(pendingReminderId))
            onPendingReminderConsumed()
        }
    }

    // A returning Discord-link OAuth deep link: make sure the Account screen is
    // on top so it can consume the code and show the result.
    LaunchedEffect(pendingDiscordLinkCode, uiState.isLoggedIn) {
        if (pendingDiscordLinkCode != null && uiState.isLoggedIn) {
            navController.navigate(Screen.Account) { launchSingleTop = true }
        }
    }

    // A new Discord-only account needs an email/password to finish onboarding.
    LaunchedEffect(uiState.discordSetup, uiState.isLoggedIn) {
        if (uiState.discordSetup != null && !uiState.isLoggedIn) {
            navController.navigate(Screen.DiscordSetup) { launchSingleTop = true }
        }
    }

    val backStackEntry by navController.currentBackStackEntryAsState()
    val currentDestination = backStackEntry?.destination
    val isTopLevel = bottomNavItems.any { item ->
        currentDestination?.hasRoute(item.route::class) == true
    }

    Scaffold(
        containerColor = BackgroundMain,
        bottomBar = {
            Column {
                if (!isOnline) {
                    Text(
                        text = stringResource(R.string.error_network),
                        style = MaterialTheme.typography.labelSmall,
                        color = ForegroundMain,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(4.dp),
                        textAlign = androidx.compose.ui.text.style.TextAlign.Center,
                    )
                }
                if (isTopLevel) {
                    NavigationBar(containerColor = BackgroundCard) {
                        bottomNavItems.forEach { item ->
                            val selected = currentDestination?.hasRoute(item.route::class) == true
                            NavigationBarItem(
                                selected = selected,
                                onClick = {
                                    navController.navigate(item.route) {
                                        popUpTo(navController.graph.findStartDestination().id) {
                                            saveState = true
                                        }
                                        launchSingleTop = true
                                        restoreState = true
                                    }
                                },
                                icon = { Icon(item.icon, contentDescription = stringResource(item.labelRes)) },
                                label = {
                                    Text(
                                        stringResource(item.labelRes),
                                        fontSize = 9.sp,
                                        maxLines = 1,
                                        softWrap = false,
                                        overflow = TextOverflow.Ellipsis,
                                    )
                                },
                                colors = NavigationBarItemDefaults.colors(
                                    selectedIconColor = AccentOrange,
                                    selectedTextColor = AccentOrange,
                                    unselectedIconColor = ForegroundMuted,
                                    unselectedTextColor = ForegroundMuted,
                                    indicatorColor = BackgroundMuted,
                                ),
                            )
                        }
                    }
                }
            }
        },
    ) { padding ->
        NavHost(
            navController = navController,
            startDestination = if (uiState.isLoggedIn) Screen.MainGraph else Screen.AuthGraph,
            modifier = Modifier.padding(padding),
            enterTransition = sandEnter,
            exitTransition = sandExit,
            popEnterTransition = sandPopEnter,
            popExitTransition = sandPopExit,
        ) {
            navigation<Screen.AuthGraph>(startDestination = Screen.Login) {
                composable<Screen.Login> {
                    LoginScreen(
                        uiState = uiState,
                        onLogin = authViewModel::loginWithEmail,
                        onRegister = authViewModel::register,
                        onLoadTimezones = authViewModel::loadTimezones,
                        onNavigateForgotPassword = {
                            authViewModel.clearFlags()
                            navController.navigate(Screen.ForgotPassword)
                        },
                        onClearError = authViewModel::clearError,
                        onDiscordUnconfigured = onDiscordUnconfigured,
                        onResendVerification = authViewModel::resendVerification,
                    )
                }
                composable<Screen.Register> {
                    RegisterScreen(
                        uiState = uiState,
                        onRegister = authViewModel::register,
                        onLoadTimezones = authViewModel::loadTimezones,
                        onRegistered = {
                            authViewModel.clearFlags()
                            navController.popBackStack()
                        },
                        onBack = { navController.popBackStack() },
                        onClearError = authViewModel::clearError,
                    )
                }
                composable<Screen.ForgotPassword> {
                    ForgotPasswordScreen(
                        uiState = uiState,
                        onSubmit = authViewModel::forgotPassword,
                        onBack = { navController.popBackStack() },
                        onClearError = authViewModel::clearError,
                    )
                }
                composable<Screen.DiscordSetup> {
                    val setup = uiState.discordSetup
                    if (setup != null) {
                        DiscordSetupScreen(
                            uiState = uiState,
                            setup = setup,
                            onComplete = authViewModel::completeDiscordSetup,
                            onLoadTimezones = authViewModel::loadTimezones,
                            onCancel = {
                                authViewModel.cancelDiscordSetup()
                                navController.popBackStack()
                            },
                            onClearError = authViewModel::clearError,
                        )
                    }
                }
            }

            navigation<Screen.MainGraph>(startDestination = Screen.Home) {
                composable<Screen.Home> {
                    HomeScreen(
                        onCreateReminder = { navController.navigate(Screen.CreateReminder) },
                        onOpenReminders = {
                            navController.navigate(Screen.Reminders) {
                                popUpTo(navController.graph.findStartDestination().id) { saveState = true }
                                launchSingleTop = true
                                restoreState = true
                            }
                        },
                        onOpenDfm = {
                            navController.navigate(Screen.Dfm) {
                                popUpTo(navController.graph.findStartDestination().id) { saveState = true }
                                launchSingleTop = true
                                restoreState = true
                            }
                        },
                        onOpenAccount = {
                            navController.navigate(Screen.Account) {
                                popUpTo(navController.graph.findStartDestination().id) { saveState = true }
                                launchSingleTop = true
                                restoreState = true
                            }
                        },
                    )
                }
                composable<Screen.Reminders> { backStackEntry ->
                    val createdFlow = backStackEntry.savedStateHandle.getStateFlow("reminder_created", false)
                    val reminderCreated by createdFlow.collectAsStateWithLifecycle()
                    RemindersScreen(
                        onCreateReminder = { navController.navigate(Screen.CreateReminder) },
                        onOpenReminder = { id -> navController.navigate(Screen.ReminderDetail(id)) },
                        showCreatedBanner = reminderCreated,
                        onCreatedBannerConsumed = { backStackEntry.savedStateHandle["reminder_created"] = false },
                    )
                }
                composable<Screen.CreateReminder> {
                    CreateReminderScreen(
                        onCreated = {
                            navController.previousBackStackEntry?.savedStateHandle?.set("reminder_created", true)
                            navController.popBackStack()
                        },
                        onBack = { navController.popBackStack() },
                    )
                }
                composable<Screen.ReminderDetail> {
                    ReminderDetailScreen(onBack = { navController.popBackStack() })
                }
                composable<Screen.Dfm> {
                    DfmScreen()
                }
                composable<Screen.DayPlanner> {
                    PlannerScreen()
                }
                composable<Screen.Account> {
                    AccountScreen(
                        onOpenApiKeys = { navController.navigate(Screen.ApiKeys) },
                        onOpenLinks = { navController.navigate(Screen.Links) },
                        onOpenAbout = { navController.navigate(Screen.About) },
                        onLogout = authViewModel::logout,
                        onAccountDeleted = authViewModel::logout,
                        pendingDiscordLinkCode = pendingDiscordLinkCode,
                        onPendingDiscordLinkConsumed = onPendingDiscordLinkConsumed,
                    )
                }
                composable<Screen.ApiKeys> {
                    ApiKeysScreen(onBack = { navController.popBackStack() })
                }
                composable<Screen.Links> {
                    LinksScreen(
                        onBack = { navController.popBackStack() },
                        onOpenTerms = { navController.navigate(Screen.Terms) },
                        onOpenPrivacy = { navController.navigate(Screen.Privacy) },
                    )
                }
                composable<Screen.Terms> {
                    TermsScreen(onBack = { navController.popBackStack() })
                }
                composable<Screen.Privacy> {
                    PrivacyScreen(onBack = { navController.popBackStack() })
                }
                composable<Screen.About> {
                    AboutScreen(onBack = { navController.popBackStack() })
                }
            }
        }
    }
}
