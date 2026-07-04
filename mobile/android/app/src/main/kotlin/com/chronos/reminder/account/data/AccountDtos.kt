package com.chronos.reminder.account.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class AccountDto(
    val id: String,
    val email: String? = null,
    val username: String? = null,
    @SerialName("email_verified") val emailVerified: Boolean = false,
    val timezone: TimezoneDto? = null,
    val identities: List<IdentityDto> = emptyList(),
    val preferences: AccountPreferencesDto? = null,
    @SerialName("created_at") val createdAt: String? = null,
)

@Serializable
data class AccountPreferencesDto(
    @SerialName("discord_send_image") val discordSendImage: Boolean? = null,
) {
    // Defaults to true (send image) when the account hasn't set a preference yet.
    val discordSendImageOrDefault: Boolean
        get() = discordSendImage ?: true
}

@Serializable
data class PreferencesRequest(
    @SerialName("discord_send_image") val discordSendImage: Boolean,
)

@Serializable
data class LinkDiscordResponse(
    val message: String = "",
    val username: String? = null,
    @SerialName("merge_required") val mergeRequired: Boolean = false,
    @SerialName("other_account_id") val otherAccountId: String? = null,
    @SerialName("discord_username") val discordUsername: String? = null,
    @SerialName("merge_token") val mergeToken: String? = null,
)

@Serializable
data class MergeRequest(
    @SerialName("merge_token") val mergeToken: String,
)

@Serializable
data class IdentityDto(
    val provider: String, // "app" | "discord"
    // For app identities the external id is the email address
    @SerialName("external_id") val externalId: String? = null,
    val username: String? = null,
    val avatar: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
)

@Serializable
data class TimezoneDto(
    val id: Int? = null,
    val name: String? = null,
    @SerialName("gmt_offset") val gmtOffset: Double = 0.0,
    @SerialName("iana_location") val ianaLocation: String,
)

@Serializable
data class TimezoneRequest(val timezone: String)

@Serializable
data class MobileIdentityRequest(
    @SerialName("device_name") val deviceName: String,
)

@Serializable
data class DiscordLinkRequest(val code: String)

@Serializable
data class AddAppIdentityRequest(
    val email: String,
    val username: String,
    val password: String,
)

@Serializable
data class UsernameRequest(val username: String)

@Serializable
data class EmailRequest(val email: String)

@Serializable
data class ChangePasswordRequest(
    @SerialName("current_password") val currentPassword: String,
    @SerialName("new_password") val newPassword: String,
)

@Serializable
data class ApiKeyDto(
    val id: String,
    val name: String,
    val scopes: String = "",
    @SerialName("created_at") val createdAt: String? = null,
    val key: String? = null, // only populated on creation
)

@Serializable
data class ApiKeysListResponse(val keys: List<ApiKeyDto> = emptyList())

@Serializable
data class CreateApiKeyRequest(val name: String)

@Serializable
data class HealthResponse(
    val status: String = "",
    val service: String = "",
    val version: String = "",
)
