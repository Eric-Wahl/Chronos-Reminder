package com.chronos.reminder.core.navigation

import kotlinx.serialization.Serializable

sealed interface Screen {

    @Serializable
    data object AuthGraph : Screen

    @Serializable
    data object Login : Screen

    @Serializable
    data object Register : Screen

    @Serializable
    data object ForgotPassword : Screen

    @Serializable
    data object DiscordSetup : Screen

    @Serializable
    data object MainGraph : Screen

    @Serializable
    data object Home : Screen

    @Serializable
    data object Reminders : Screen

    @Serializable
    data object CreateReminder : Screen

    @Serializable
    data class ReminderDetail(val id: String) : Screen

    @Serializable
    data object Dfm : Screen

    @Serializable
    data object DayPlanner : Screen

    @Serializable
    data object Account : Screen

    @Serializable
    data object ApiKeys : Screen

    @Serializable
    data object Links : Screen

    @Serializable
    data object Terms : Screen

    @Serializable
    data object Privacy : Screen

    @Serializable
    data object About : Screen
}
