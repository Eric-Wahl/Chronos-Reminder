[![Chronos Reminder](img/Chronos%20Reminder.png)](https://github.com/Eric-Wahl/Chronos-Reminder)

---

![Version](https://img.shields.io/badge/Version-1.0.2-green)
[![Website](https://img.shields.io/badge/Website-chronosrmd.com-Orange?labelColor=Orange&style=flat&logo=google-chrome&link=https://chronosrmd.com)](https://chronosrmd.com)
[![Documentation](https://img.shields.io/badge/Documentation-Docs-orange?labelColor=Red&style=flat&logo=read-the-docs&link=https://docs.chronosrmd.com)](https://docs.chronosrmd.com)
[![Join Discord](https://img.shields.io/badge/Join%20Discord-Discord-green?labelColor=Blue&style=flat&logo=discord&link=https://discord.gg/m3MsM922QD)](https://discord.gg/m3MsM922QD)
[![Invite the Bot](https://img.shields.io/badge/Invite%20the%20Bot-on%20Discord-blue?labelColor=Blue&style=flat&logo=discord&link=https://discord.com/oauth2/authorize?client_id=955923021732913254&permissions=2416127056&scope=bot)](https://discord.com/oauth2/authorize?client_id=955923021732913254&permissions=2416127056&scope=bot)
[![License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/Eric-Wahl/Chronos-Reminder)

---

**Chronos Reminder** is your **new personal time assistant**, easy to integrate into your daily life.

⌛️ - Go check the live webapp here: [chronosrmd.com](https://chronosrmd.com) !

⏳ - This project is the successor of the previous [Kairos Discord Reminder Bot](https://github.com/Eric-Wahl/Kairos-Bot-Reminder) !

> [!TIP]  
> This project is still under active development, so expect new features and improvements regularly !
> This project is in its first version. Feel free to contribute or suggest features!

## Key Features

- **Easy-to-use Discord Bot**: Quickly set reminders directly using natural language commands.
- **Web Dashboard**: Manage your reminders and settings through a user-friendly web interface.
- **Multi-language Support**: Available in English, Spanish, and French to cater to a diverse user base.
- **Timezone Awareness**: Set reminders based on your local timezone for accurate notifications.
- **Self-Hosting Option**: Deploy Chronos Reminder on your own server for complete control over your data.
- **Free and Open Source**: Completely free to use and modify under the MIT License.
- **Lightweight and Efficient**: Designed to run smoothly without consuming excessive resources, while being reliable, with real-time reminder management.
- **Recurring Reminders**: Set up reminders that repeat at specified intervals (daily, weekly, monthly, yearly).
- **Multiple Destination Support**: Receive reminders via Discord DMs, Discord Channels and Webhooks (more to come).

## Documentation

Comprehensive documentation is available at [docs.chronosrmd.com](https://docs.chronosrmd.com) to help you get started, configure, and make the most out of Chronos Reminder.

You can contact me from the [website](https://chronosrmd.com) or join the [Discord community](https://discord.gg/m3MsM922QD) for support and discussions.

## Technologies

Here you'll find the main technologies used in this project

| Component      | Technology/Framework | Version         |
| -------------- | -------------------- | --------------- |
| Backend        | Go (Golang)          | 1.24.4          |
| Bot            | DiscordGo            | v0.29.0         |
| Frontend       | Node.js & React      | 24.3.0 & 19.1.1 |
| Database       | PostgreSQL           | 16.0            |
| Cache          | Redis                | 8.0             |
| Uptime Monitor | UptimeKuma           | latest          |

## Project structure

```
chronos-reminder/
├── internal/ # Backend source code
│ ├── api/ # API related code
│ ├── bot/ # Discord bot related code
│ ├── config/ # Configuration files
│ ├── database/ # Database interaction code (PostgreSQL and Redis)
│ ├── services/ # Business logic and services
│ ├── engine/ # Main queue and scheduler engine
│ ├── dispatchers/ # Notification dispatchers (Discord, Email, etc.), working with the engine
│ ├── tests/ # Unit and integration tests
│ └── docs/ # Swagger documentation
├── web/ # Frontend source code
│ ├── public/ # Public assets
│ ├── src/ # React source code
│ ├── ├── components/ # Common and specific components
│ ├── ├── i18n/ # Internationalization files
│ ├── ├── hooks/ # Custom React hooks
│ ├── ├── lib/ # Utility functions and libraries
│ ├── ├── pages/ # React pages
│ ├── ├── services/ # API interaction services
│ ├── └── types/ # TypeScript type definitions
│ ├── Dockerfile # Dockerfile for building the web frontend
│ ├── nginx.conf # Nginx configuration for the Docker setup
│ └── .env # Environment variables for web
└── Dockerfile # Dockerfile for building the backend and bot
```

## Next Features

- **Create new reminders using natural language**: Just type what you want to be reminded of and when, and Chronos will handle the rest.
- **More notification methods**: Adding support for email, SMS, and push notifications.

## Development & Contributions

Contributuions encouraged and very welcome, however some rules and guidelines must be followed!

Contributuions encouraged and very welcome, however some rules and guidelines must be followed!

### General Guidelines

- The project is versioned according to [Semantic Versioning](https://semver.org/).
- When writing your commit messages, please follow the [Angular commit message](https://gist.github.com/brianclements/841ea7bffdb01346392c).
- Pull requests should be made against the `develop` branch, so please make sure you check out the `develop` branch.
- Pull requests should include tests and documentation as appropriate.
- When opening a pull request, if possible, attach a screenshot or GIF of the changes.

### Feature Requests

Open a new discussion with the `feature request` tag and describe the feature you would like to see implemented. If you have a screenshot or GIF of the feature, please attach it to the discussion.

### Have a Question?

If you need any help, have a question, or just want to discuss something related to the project, please feel free to join the [Discord community](https://discord.gg/m3MsM922QD) or open a new discussion.

## Known Issues

/

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributors

Thanks to the following people who have contributed to this project:

- Eric-Wahl - [GitHub](https://github.com/Eric-Wahl)
