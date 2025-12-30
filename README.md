Validator Dashboard (minimal)

Run the API server:

```bash
go run ./cmd/server
```

POST /validator with JSON body:

{
  "validatorId": [12345]
}

Example response (successful):

```json
{
  "successes": {
    "12345": {
      "overview": {
        "id": 12345,
        "slashed": false,
        "status": "active",
        "withdrawal_credentials_type": "0x00",
        "withdrawal_credentials": "0x...",
        "activation_epoch": 123456,
        "exit_epoch": 0,
        "current_balance_raw": "32000000000",
        "current_balance_human": "0.032000000",
        "online": true,
        "fetched_at": "2025-12-30T12:00:00Z"
      },
      "rewards": {
        "total_raw": "120000000000",
        "total_human": "0.000120000000"
      },
      "performance": {
        "beacon_score": 0.98
      }
    }
  },
  "errors": {}
}
```
