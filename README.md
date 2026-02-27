# Orderly Network Data Source Plugin

A PlusEV data source plugin for Orderly Network, providing access to perpetual futures market data.

## Features

- **Market Data**: Access to all available Orderly Network perpetual futures markets
- **Historical Data**: Fetch historical OHLCV (candlestick) data using TradingView-compatible API
- **Real-time Streaming**: WebSocket-based real-time OHLCV data streams
- **Multiple Timeframes**: Support for 1m, 5m, 15m, 30m, 1h, 4h, 12h, 1d, 1w, 1M
- **Authentication**: Optional ED25519-based authentication for enhanced rate limits
- **Order Placement**: Support for batch order creation and algo orders (TP/SL, bracket, trailing stop)

## Configuration

The plugin requires the following configuration:

### Required
- **Account ID**: Your Orderly Network Account ID

### Optional (for enhanced features)
- **API Key**: Your Orderly Network API Key
- **Private Key**: Your ED25519 private key (base58 encoded)
- **Use Testnet**: Enable to connect to Orderly testnet instead of mainnet

## Supported Endpoints

### Public Endpoints
- `GET /v1/public/info` - Available trading symbols
- `GET /v1/tv/history` - Historical OHLCV data (TradingView format)

### WebSocket Streams
- Market Data: `wss://ws-evm.orderly.org/ws/stream/{account_id}`
- Testnet: `wss://testnet-ws-evm.orderly.org/ws/stream/{account_id}`

## API Limits

- Public endpoints: 10 requests per second per IP
- TradingView history: 10 requests per second per IP
- WebSocket connections: Account-based limits

## Building

```bash
chmod +x build.sh
./build.sh
```

This will create `orderly-datasource.wasm` ready for use with PlusEV.

## Supported Markets

The plugin supports all Orderly Network perpetual futures markets with symbols in the format `PERP_BASE_QUOTE` (e.g., `PERP_BTC_USDC`).

## Timeframes

| Format | Resolution | WebSocket Topic |
|--------|------------|-----------------|
| 1m     | 1          | kline_1m        |
| 5m     | 5          | kline_5m        |
| 15m    | 15         | kline_15m       |
| 30m    | 30         | kline_30m       |
| 1h     | 60         | kline_1h        |
| 4h     | 240        | kline_4h        |
| 12h    | 720        | kline_12h       |
| 1d     | 1D         | kline_1d        |
| 1w     | 1W         | kline_1w        |
| 1M     | 1M         | kline_1M        |

## License

MIT License - see LICENSE file for details.

## Links

- [Orderly Network](https://orderly.network/)
- [Orderly API Documentation](https://orderly.network/docs)
- [PlusEV Terminal](https://plusev.io/)