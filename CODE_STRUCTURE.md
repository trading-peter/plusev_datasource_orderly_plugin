# Orderly Plugin Code Structure

The Orderly plugin code has been reorganized into logical files for better maintainability:

## Files Overview

### 1. `client.go` - Core Client
**Purpose**: Core client struct, constructor, credentials management, and basic configuration
**Contents**:
- `Client` struct definition
- `NewClient()` constructor
- `SetCredentials()` method
- `GetName()` and `GetConfigFields()` methods

### 2. `types.go` - Type Definitions
**Purpose**: All type definitions, structs, and constants
**Contents**:
- Constants: `OrderlyMaxLimit`, `DefaultLimit`
- API response types: `OrderlyResponse`, `Symbol`, `SymbolsResponse`, `TradingViewHistoryResponse`
- WebSocket types: `WSSubscribeMessage`, `WSResponse`, `WSKlineUpdate`, `WSKlineData`

### 3. `markets.go` - Market Operations
**Purpose**: Market discovery, timeframes, and utility functions
**Contents**:
- `GetMarkets()` - Fetches available trading markets
- `GetTimeframes()` - Returns supported timeframes
- `countDecimalsFromString()` and `countDecimals()` - Decimal precision utilities
- `convertTimeframeToTradingViewResolution()` - Timeframe conversion for TradingView API
- `convertTimeframeToWebSocketTopic()` - Timeframe conversion for WebSocket

### 4. `ohlcv.go` - OHLCV Data
**Purpose**: Historical candlestick data fetching with dynamic parameter handling
**Contents**:
- `GetOHLCV()` - Enhanced with the new parameter logic you requested:
  - Limit defaults to 100
  - If start, end nil → serve "limit" candles back from now
  - If start not nil, end nil → serve "limit" candles forward from start
  - If start nil, end not nil → serve "limit" candles back starting at end
  - If start not nil, end not nil → serve candles from start to end (capped by 1000)

### 5. `streaming.go` - WebSocket Streaming
**Purpose**: All WebSocket-related functionality for real-time data
**Contents**:
- `PrepareStream()` - Sets up WebSocket connections
- `HandleStreamMessage()` - Processes incoming messages
- `HandleConnectionEvent()` - Handles connection state changes

### 6. `auth.go` - Authentication
**Purpose**: Authentication and authorization functionality
**Contents**:
- `isAuthenticated()` - Checks if credentials are configured
- `addAuthHeaders()` - Adds ED25519 signature headers for private endpoints

## Benefits

1. **Better Organization**: Each file has a single responsibility
2. **Easier Maintenance**: Changes to specific functionality are isolated
3. **Improved Readability**: Smaller files are easier to navigate and understand
4. **Reduced Complexity**: The original 947-line file is now split into manageable chunks
5. **Better Collaboration**: Multiple developers can work on different aspects without conflicts

## Code Quality

- All files compile without errors
- Go formatting (`go fmt`) passes on all files
- Maintains all existing functionality
- No breaking changes to the public API

The refactoring maintains full backward compatibility while significantly improving code organization and maintainability.