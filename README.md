# Telegram Bot for Community Message Moderation

This project implements a Telegram bot that allows users to submit messages for publication in a community group. The bot sends these messages to an administrator for moderation, who can approve or reject them. Approved messages are then forwarded to the group, while rejected messages receive a notification back to the user.

## Project Structure

```
telegram-bot
├── cmd
│   └── main.go          # Entry point of the application
├── internal
│   ├── bot
│   │   ├── bot.go      # Main bot logic
│   │   └── handlers.go  # Message handlers
│   ├── config
│   │   └── config.go    # Configuration management
│   └── models
│       └── message.go    # Data structures for messages
├── go.mod                # Module definition
├── go.sum                # Dependency checksums
└── README.md             # Project documentation
```

## Setup Instructions

1. **Clone the repository:**
   ```
   git clone https://github.com/yourusername/telegram-bot.git
   cd telegram-bot
   ```

2. **Install dependencies:**
   Ensure you have Go installed, then run:
   ```
   go mod tidy
   ```

3. **Configure the bot:**
   Update the configuration file located in `internal/config/config.go` with your bot token, admin chat ID, and group chat ID.

4. **Run the bot:**
   Execute the following command to start the bot:
   ```
   go run cmd/main.go
   ```

## Usage

- Users can start a chat with the bot and send messages they wish to publish.
- The bot will forward these messages to the designated admin for approval.
- Admins can approve or reject messages using the provided buttons.
- Approved messages will be sent to the community group, while rejected messages will notify the user of the decision.

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue for any enhancements or bug fixes.

## License

This project is licensed under the MIT License. See the LICENSE file for details.