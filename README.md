## Overview

This is a simple chat application that allows users to send and receive messages in real-time using WebSockets. The application supports functionalities like user authentication, creating/joining/leaving chat rooms, sending direct messages, and broadcasting messages to all users in a chat room. It also includes a command-line interface (CLI) for interaction and supports saving chat history and user data to files.

## Features

- **Real-time communication** using WebSockets.
- **User authentication**.
- **Chat rooms**: Users can create, join, and leave chat rooms.
- **Direct messages**: Users can send private messages to other users.
- **Persistent storage**: Chat history and user data are saved to JSON files.
- **Error handling** for various scenarios such as connection errors.
- **Command-line interface** for user interaction.

## Technologies Involved

- **Go**: The programming language used to build the application.
- **Gorilla WebSocket**: A Go package used to implement WebSocket connections.
- **Cobra**: A library for creating powerful CLI applications in Go.
- **Promptui**: A library for creating interactive prompts in CLI applications.
- **Sync**: For handling concurrent access to shared resources.
- **JSON**: For saving and loading user data and chat history.
- **Docker**: For containerizing the application (still in troubleshooting).

## Code Structure

- `main.go`: The main entry point of the application.
- `User`, `ChatRoom` structs: Represent users and chat rooms.
- `serverCmd` and `clientCmd`: Cobra commands for starting the server and client respectively.
- `handleConnections`, `authenticateUser`, `handleMessage`: Functions to handle WebSocket connections, authentication, and message handling.
- `createRoom`, `joinRoom`, `leaveRoom`, `broadcastMessage`, `sendDirectMessage`: Functions for managing chat rooms and messages.
- `saveData`, `loadData`, `saveToFile`, `loadFromFile`: Functions for saving and loading data to/from JSON files (chat.json, users.json).
- `promptUser`, `registerUser`: Functions for CLI interaction and user registration.

## How to Run the Demo

### Prerequisites

- **Go** installed on your machine. You can download it from [here](https://golang.org/dl/).
- **Git** installed on your machine to clone the repository.
- **wscat** installed. You can install it globally via npm:

  ```sh
  npm install -g wscat
  ```

### Steps

1. **Clone the repository**:

   ```sh
   git clone <repository-url>
   cd <repository-directory>
   ```

2. **Build the application**:

   ```sh
   go mod download

   go build -o chatapp main.go
   ```

3. **Run the server**:

   ```sh
   ./chatapp server
   ```

   The server will start and listen for WebSocket connections on `ws://localhost:8080/ws`.

4. **Test the WebSocket connection** (optional):

   Open another terminal window and run:

   ```sh
   wscat -c ws://localhost:8080/ws
   ```

   This will connect to the WebSocket server, and you can manually send messages for testing.

5. **Run the client**:

   Open another terminal window and run:

   ```sh
   ./chatapp client
   ```

   Follow the prompts to register, login, create/join chat rooms, and send messages.

### Using the CLI

- **Register**: Register a new user with a username and password.
- **Login**: Login with your registered username and password.
- **Create Room**: Create a new chat room.
- **Join Room**: Join an existing chat room.
- **Leave Room**: Leave a chat room.
- **Send Message**: Send a message to a chat room.
- **Direct Message**: Send a private message to another user.
- **Quit**: Exit the application.

## Known Issues

### Docker

Currently, Docker support is under development and troubleshooting. The plan is to containerize the application using Docker and provide a `Dockerfile` and `docker-compose.yml` file for easy setup and deployment. Once the issues are resolved, an updated version of this documentation will include instructions on how to use Docker to run the application.

### User Registration and Login

There are known issues with the user registration and login process:

- Users may encounter errors during registration or login.
- These issues are being actively investigated and addressed.

### Chat Room Functionality

The chat room functionality is still buggy:

- Users can create and join chat rooms.
- However, after joining or creating a room, users may experience issues when trying to chat like in a common chat application view.
- This is currently a work in progress, and improvements are being made to ensure smooth chatting within rooms.

## Troubleshooting

### Common Issues

1. **WebSocket connection fails**:

   - Ensure the server is running and listening on the correct port.
   - Check for network issues that may be blocking WebSocket connections.

2. **Authentication fails**:

   - Verify that the username and password are correct.
   - Ensure that the user data is correctly saved and loaded from the `users.json` file.

3. **Messages not being delivered**:
   - Check if the user is correctly joined to the chat room.
   - Ensure there are no issues with the WebSocket connection.
