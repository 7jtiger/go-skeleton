# Content Controller Function Summary

## ContentController

### Structure and Initialization Functions
- `NewContentController()`: Constructor function that initializes a ContentController instance and sets up necessary dependencies

### Basic Service Functions
- `GetTestContent()`: API endpoint function that tests whether the content controller is operating normally

---

## ChatRoomListController

### Structure and Initialization Functions
- `NewChatRoomListController()`: Constructor function that initializes a ChatRoomListController instance and sets up Redis DB dependencies

### Chat Room List Management Functions
- `GetChatRoomList()`: Function that retrieves user's chat room list sorted by last activity time and priority
- `JoinChatRoom()`: Function that adds a user to a chat room and includes it in the chat room list
- `LeaveChatRoom()`: Function that removes a user from a chat room and deletes it from the chat room list

### Chat Room Event Handling Functions
- `HandleChatRoomEvent()`: Function that moves chat rooms to the top based on events such as messages, mentions, urgent messages, read status, etc.
- `MarkAsRead()`: Function that marks messages as read by setting the unread message count to 0 for a specific chat room

### Chat Room Statistics Functions
- `GetChatRoomStats()`: Function that retrieves statistical information such as total number of chat rooms and unread message count for a user

---

## Overall Structure Summary

### ContentController (1 function)
- Provides basic test endpoint

### ChatRoomListController (7 functions)
- **Initialization**: Controller constructor (1)
- **List Management**: Retrieve, join, leave (3)  
- **Event Handling**: Event handling, mark as read (2)
- **Statistics**: Chat room statistics retrieval (1)


