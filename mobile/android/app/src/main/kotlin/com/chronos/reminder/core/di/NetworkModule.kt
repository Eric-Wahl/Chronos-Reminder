package com.chronos.reminder.core.di

import com.chronos.reminder.account.data.AccountApi
import com.chronos.reminder.auth.data.AuthApi
import com.chronos.reminder.core.network.ApiClient
import com.chronos.reminder.dfm.data.DfmApi
import com.chronos.reminder.notifications.FcmApi
import com.chronos.reminder.planner.data.PlannerApi
import com.chronos.reminder.reminders.data.DiscordApi
import com.chronos.reminder.reminders.data.RemindersApi
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import kotlinx.serialization.json.Json
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {

    @Provides
    @Singleton
    fun provideJson(apiClient: ApiClient): Json = apiClient.json

    @Provides
    @Singleton
    fun provideAuthApi(apiClient: ApiClient): AuthApi = apiClient.retrofit.create(AuthApi::class.java)

    @Provides
    @Singleton
    fun provideRemindersApi(apiClient: ApiClient): RemindersApi = apiClient.retrofit.create(RemindersApi::class.java)

    @Provides
    @Singleton
    fun provideDiscordApi(apiClient: ApiClient): DiscordApi = apiClient.retrofit.create(DiscordApi::class.java)

    @Provides
    @Singleton
    fun provideDfmApi(apiClient: ApiClient): DfmApi = apiClient.retrofit.create(DfmApi::class.java)

    @Provides
    @Singleton
    fun providePlannerApi(apiClient: ApiClient): PlannerApi = apiClient.retrofit.create(PlannerApi::class.java)

    @Provides
    @Singleton
    fun provideAccountApi(apiClient: ApiClient): AccountApi = apiClient.retrofit.create(AccountApi::class.java)

    @Provides
    @Singleton
    fun provideFcmApi(apiClient: ApiClient): FcmApi = apiClient.retrofit.create(FcmApi::class.java)
}
