# AFI SDK

> SDK de nivel productivo para ejecutar swaps de tokens en redes EVM a través del [Protocolo AFI](https://afi.run).

Construye interfaces de swap, bots de trading, herramientas analíticas e
indexers sin reimplementar descubrimiento de rutas, matemática de slippage,
flujo de allowance, buffer de gas, decodificación de revert ni parsing de eventos.

| | |
|---|---|
| **Lenguajes** | TypeScript (Node.js 18+) · Go 1.21+ |
| **Redes (cotización)** | Base · BSC · Arbitrum · Ethereum · Unichain |
| **Redes (ejecución)** | Todas las anteriores (chain ID detectado desde el RPC) |
| **Traducciones** | [English](./README.md) · [Português (BR)](./README.pt-BR.md) |
| **Licencia** | MIT |

---

## Por qué este SDK

- **Swap en una llamada** — `client.swap()` encadena cotización → verificación de saldo → approve → simulate → submit → wait.
- **Flujo por etapas** — cada paso también se expone individualmente para control granular de UI.
- **Seguro por defecto** — allowances exactos, `minOut` aplicado on-chain, simulación antes del broadcast.
- **Cotizaciones multi-chain** — cotiza en 5 redes EVM desde un único client.
- **Ergonomía operativa** — health checks, logs estructurados, serialización JSON, lecturas vía multicall, buffer de gas configurable, confirmaciones, timeouts.

---

## ¿Qué quieres hacer?

Un mapa rápido desde tu rol hasta el entrypoint correcto. Cada elemento de la
derecha enlaza a una sección de este documento; los encoders ya están
disponibles hoy, y los wrappers `client.*` de alto nivel son azúcar sintáctico
encima de ellos.

| Rol | Objetivo | Usa |
|---|---|---|
| Usuario final | Intercambiar tus propios tokens | `client.swap()` o `client.quote().execute()` |
| Operador | Swap por 1 usuario pre-aprobado | `client.swapFor({ user, tokenIn, tokenOut, amountIn })` |
| Operador | Batch swap para varios usuarios | `client.batchSwapFor([{ user, ... }, ...])` |
| Operador (arb) | Ciclo de arbitraje con flash loan | `client.executeNMRArbitrage({ asset, amount, params })` |
| Operador | Arb financiada por el usuario (NMR.loan) | `client.nmrLoanArbitrage({ user, asset, amount, ... })` |
| Operador | Retirar ganancia del NMR | `client.sweepNMRProfit({ asset, amount })` |
| Owner | Pause / unpause del router | `client.adminPause()` / `client.adminUnpause()` |
| Owner | Cambiar la tarifa global | `client.adminSetFeeBps(bps)` |
| Owner | Override de tarifa por usuario | `client.adminSetUserFeeBps(user, bps)` |
| Owner | Agregar regla de validación | `client.adminAddRule(rule)` |
| Inspector | Verificación de deploy | `client.verifyDeployment(chainId)` |
| Indexer | Parsear eventos | `parseSwapExecuted(logs)`, `parseFlashLoanExecuted(logs)`, ... |

---

## Tabla de contenidos

- [Requisitos](#requisitos)
- [Instalación](#instalación)
- [Inicio rápido](#inicio-rápido)
- [Conceptos centrales](#conceptos-centrales)
- [Referencia de la API](#referencia-de-la-api)
  - [Construcción del client](#construcción-del-client)
  - [Operaciones de lectura](#operaciones-de-lectura)
  - [Builder de cotización](#builder-de-cotización)
  - [Operaciones de escritura (requieren signer)](#operaciones-de-escritura-requieren-signer)
  - [Utilidades de transacción](#utilidades-de-transacción)
  - [Configuración](#configuración)
- [Helpers](#helpers)
- [Logs y diagnóstico](#logs-y-diagnóstico)
- [Manejo de errores](#manejo-de-errores)
- [Modelo de seguridad](#modelo-de-seguridad)
- [Recetas](#recetas)
- [Flujos de operador](#flujos-de-operador)
- [Admin / gobernanza](#admin--gobernanza)
- [Indexación de eventos](#indexación-de-eventos)
- [Builders de step por DEX](#builders-de-step-por-dex)
- [Endpoints HTTP del quoter](#endpoints-http-del-quoter)
- [Guía de migración](#guía-de-migración)
- [Redes y constantes](#redes-y-constantes)
- [Directorio de ejemplos](#directorio-de-ejemplos)
- [Desarrollo](#desarrollo)
- [Licencia](#licencia)

---

## Requisitos

| Runtime    | Mínimo   | Recomendado |
|------------|----------|-------------|
| Node.js    | 18.x     | 20.x LTS    |
| TypeScript | 5.0      | última      |
| Go         | 1.21     | 1.22+       |

También necesitas un endpoint RPC HTTP para cada red en la que vayas a leer o
ejecutar. Proveedores públicos (Ankr, Alchemy, Infura, drpc, …) funcionan en
desarrollo; **usa un plan pago (o tu propio nodo) en producción** para evitar
timeouts del quoter y reverts por rate-limit.

---

## Instalación

### TypeScript / Node.js

```bash
npm install @afi-run/sdk     # o: pnpm add @afi-run/sdk · yarn add @afi-run/sdk
```

Hasta que el paquete se publique en npm:

```bash
git clone https://github.com/afi-run/sdk.git
npm install ./sdk/nodejs
```

### Go

```bash
go get github.com/afi-run/sdk/go
```

```go
import afi "github.com/afi-run/sdk/go"
```

---

## Inicio rápido

### TypeScript — cotización de solo lectura

```typescript
import { AfiClient, NETWORK, formatUnits } from "@afi-run/sdk"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/TU_CLAVE",
})

const quote = await client
  .quote(
    "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",  // USDC
    "0x4200000000000000000000000000000000000006",  // WETH
    "1000",
  )
  .slippage(0.5)
  .network(NETWORK.BASE)
  .get()

console.log(`Estimado:    ~${quote.amountOut} WETH`)
console.log(`Mínimo:      ${quote.minOut} WETH`)
console.log(`Hops:        ${quote.hops.length}`)
console.log(`Creado en:   ${new Date(quote.createdAt).toISOString()}`)
```

### TypeScript — swap en una llamada

```typescript
client.connect("0xTU_CLAVE_PRIVADA")

const result = await client
  .quote(USDC, WETH, "500")
  .slippage(0.5)
  .execute({ confirmations: 1 })

console.log(`Tx:         ${client.txUrl(result.txHash)}`)
console.log(`Recibido:   ${formatUnits(result.amountOut, 18)} WETH`)
console.log(`Gas usado:  ${result.gasUsed}`)
```

### Go — cotización de solo lectura

```go
import (
    "context"
    "fmt"
    "log"

    afi "github.com/afi-run/sdk/go"
    "github.com/ethereum/go-ethereum/common"
)

func main() {
    client, err := afi.NewClient(afi.Config{
        RPCURL: "https://rpc.ankr.com/base/TU_CLAVE",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()
    usdc := common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")

    quote, err := client.GetQuote(ctx,
        afi.From(usdc, afi.WETH, "1000"),
        afi.WithSlippage(0.5),
        afi.OnNetwork(afi.NetworkBase),
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Estimado:    ~%s WETH\n", quote.AmountOut)
    fmt.Printf("Mínimo:      %s WETH\n", quote.MinOut)
    fmt.Printf("Hops:        %d\n", len(quote.Hops))
}
```

### Go — swap en una llamada

```go
client.Connect("TU_CLAVE_PRIVADA")

result, err := client.Swap(ctx,
    afi.From(usdc, afi.WETH, "500"),
    afi.WithSlippage(0.5),
)
if err != nil {
    log.Fatal(err)
}
url, _ := client.TxURL(result.TxHash.Hex())
fmt.Printf("Tx:        %s\n", url)
fmt.Printf("Recibido:  %s WETH\n", afi.FormatUnits(result.AmountOut, 18))
```

---

## Conceptos centrales

### Ciclo de vida del swap

Cada swap pasa por cinco etapas. `executeSwap(quote)` corre las etapas 2–5 de
forma atómica; cada una también se expone individualmente para flujos paso a paso.

```
1. Cotización     ─ POST /quoter — calcula ruta, slippage, minOut
2. Saldo          ─ ERC20.balanceOf(owner) ≥ amountIn
3. Approve        ─ ERC20.approve(AFI, amountInWei)        (se omite si ya hay allowance suficiente)
4. Simulación     ─ eth_call AFI.swap(...)                 (falla rápido si revertiría)
5. Envío + espera ─ broadcast y espera de confirmaciones
```

### Modo lectura vs modo signer

El client tiene dos modos según haya o no clave privada configurada:

- **Lectura** — `quote`, `tokenInfo`, `getBalance`, `getEthBalance`,
  `getAllowance`, `hasAllowance`, `getFeeBps`, `chainId`, `detectNetwork`,
  `health`, `txUrl`, `addressUrl`.
- **Signer** (agrega) — `approve`, `simulate`, `submitSwap`, `executeSwap`,
  `swap`, `estimateSwapCost`.

Los métodos de lectura siguen disponibles tras `connect()`. Los de escritura
lanzan `NoSignerError` cuando no hay clave privada.

### Modelo de buffer de gas

Todas las transacciones de escritura (approve + swap) multiplican el resultado
de `eth_estimateGas` por `(1 + gasBufferPercent / 100)`. El predeterminado es
**+15 %**. Configura con `gasBufferPercent` en la construcción del client, o
sobrescribe en runtime con `setGasBufferPercent(n)`. Pasa `0` para desactivar.

El buffer solo afecta el gas enviado a `writeContract` / `SendTransaction`,
nunca el precio (el `maxFeePerGas` siempre es `baseFee * 2 + tip`).

### Slippage y garantía de `minOut`

Toda `Quote` lleva `minOutWei` — el mínimo que el router AFI acepta on-chain.
El contrato revierte si la ejecución fuera a entregar menos, así que el usuario
nunca recibe menos que ese valor. El SDK rechaza cotizaciones con
`minOutWei = 0`.

Slippage se expresa en porcentaje (`0.5` = 0,5 %) y lo aplica el quoter. Usa el
helper [`calculateMinOut`](#calculadora-de-slippage) si necesitas derivarlo del
lado del cliente.

### Builder vs functional options

- **TypeScript** — `client.quote(...)` devuelve un `QuoteBuilder` fluido.
  Encadena `.slippage()`, `.maxHops()`, `.network()`, etc., y termina con `.get()` o `.execute()`.
- **Go** — `client.GetQuote(ctx, opts...)` recibe functional options
  (`afi.From`, `afi.WithSlippage`, `afi.OnNetwork`, …).

Ambos exponen la misma configuración; elige el estilo que ya use tu codebase.

---

## Referencia de la API

### Construcción del client

#### TypeScript

```typescript
new AfiClient(config: AfiConfig)

interface AfiConfig {
  rpcUrl:             string             // requerido — RPC de la red de ejecución
  privateKey?:        Hex                // opcional — habilita modo signer
  gasBufferPercent?:  number             // predet.: 15 — % sobre estimateGas
  logger?:            Logger             // opcional — callback de diagnóstico
}
```

#### Go

```go
afi.NewClient(cfg afi.Config) (*afi.Client, error)

type Config struct {
    RPCURL           string  // requerido
    PrivateKey       string  // opcional — hex con o sin 0x
    GasBufferPercent uint    // predet.: 15 — cero usa el default; SetGasBufferPercent(0) desactiva
    Logger           Logger  // opcional
}
```

`Close()` (Go) cierra la conexión RPC subyacente.

---

### Operaciones de lectura

| Método | Retorna | Descripción |
|---|---|---|
| `getTokens(network?)` / `GetTokens(ctx, network?)` | `Token[]` | Tokens activos. Cache por red. |
| `findToken(symbol, network?)` / `FindToken(ctx, symbol, network?)` | `Token \| null` | Búsqueda case-insensitive. Usa el cache. |
| `clearTokensCache(network?)` / `ClearTokensCache(network?)` | `void` | Invalida el cache (todas las redes o una). |
| `getFeeBps()` / `GetFeeBps(ctx)` | `number` / `uint16` | Tarifa actual del protocolo en el contrato. |
| `tokenInfo(token, owner?)` / `TokenInfo(ctx, token, owner)` | `TokenInfo` | symbol/name/decimals (+ balance/allowance) en **un multicall**. |
| `tokenInfoBatch(tokens, owner?)` / `TokenInfoBatch(ctx, tokens, owner)` | `TokenInfo[]` | Lo mismo para N tokens en un único multicall. |
| `getBalance(token, owner?)` / `GetBalance(ctx, token, owner?)` | `bigint` / `*big.Int` | Saldo ERC-20. |
| `getEthBalance(owner?)` / `GetETHBalance(ctx, owner?)` | `bigint` / `*big.Int` | Saldo de ETH nativo. |
| `getAllowance(token, owner?)` / `GetAllowance(ctx, token, owner?)` | `bigint` / `*big.Int` | Cuánto puede gastar el router AFI por `owner`. |
| `hasAllowance(token, amount, owner?)` / `HasAllowance(ctx, token, amount, owner?)` | `boolean` | Conveniencia: `getAllowance >= amount`. |
| `chainId()` / `ChainID(ctx)` | `number` / `*big.Int` | Chain ID leído del RPC (cacheado). |
| `detectNetwork()` / `DetectNetwork(ctx)` | `Network \| null` | Mapea el chain ID a una `Network` conocida. |
| `health()` / `Health(ctx)` | `HealthCheck` | Probe paralelo de RPC + API. |
| `estimateSwapCost(quote)` / `EstimateSwapCost(ctx, quote)` | `SwapCostEstimate` | Proyecta el costo sin enviar tx. **Requiere signer.** |

`owner` omitido usa la wallet conectada. En Go pasa `common.Address{}` para el
mismo efecto. `TokenInfo` en TS acepta `"self"` como atajo.

#### Token

```typescript
interface Token {
  address:  Address     // 0x… 20 bytes
  symbol:   string      // p.ej. "USDC"
  decimals: number      // p.ej. 6
  active:   boolean     // false ⇒ deprecated/pausado
}
```

#### TokenInfo

```typescript
interface TokenInfo {
  address:    Address
  symbol:     string
  name:       string
  decimals:   number
  owner?:     Address    // solo cuando se pasó owner
  balance?:   bigint     // saldo ERC-20 del owner
  allowance?: bigint     // allowance concedida al AFI por el owner
}
```

#### HealthCheck

```typescript
interface HealthEndpoint {
  ok:          boolean
  durationMs:  number
  detail?:     string    // "chainId=8453" en el RPC, "ok" o "HTTP 503" en la API
  error?:      unknown
}

interface HealthCheck {
  rpc: HealthEndpoint
  api: HealthEndpoint
}
```

#### SwapCostEstimate

```typescript
interface SwapCostEstimate {
  gas:           bigint   // eth_estimateGas crudo
  gasWithBuffer: bigint   // gas * (1 + gasBufferPercent/100)
  gasPriceWei:   bigint   // maxFeePerGas que el SDK usaría = baseFee * 2 + tip
  totalWei:      bigint   // gasWithBuffer * gasPriceWei
  totalEth:      string   // totalWei formateado en ETH (18 decimales)
}
```

---

### Builder de cotización

#### TypeScript

```typescript
client.quote(tokenIn: Address | Token, tokenOut: Address | Token, amountIn: string): QuoteBuilder
```

| Método            | Predet.     | Descripción |
|-------------------|-------------|-------------|
| `.slippage(v)`    | `0.5`       | Tolerancia de slippage en % |
| `.maxHops(n)`     | `2`         | Máximo de hops |
| `.network(n)`     | `BASE`      | Red objetivo |
| `.priceBase(s)`   | —           | Llena `tokenInBasePrice` / `tokenOutBasePrice` |
| `.dexs(...)`      | —           | Restringe DEXes |
| `.blockNumber(n)` | `"latest"`  | Cotizar contra un bloque específico |
| `.rpcUrls(...)`   | RPC client  | Sobrescribe los endpoints que usa el quoter |
| `.get()`          | —           | Trae y devuelve un `Quote` |
| `.execute(opts?)` | —           | Trae + ejecuta. Requiere signer. |

#### Go

```go
client.GetQuote(ctx context.Context, opts ...QuoteOption) (*Quote, error)
client.Swap(ctx context.Context, opts ...QuoteOption) (*SwapResult, error)
```

| Option                   | Predet.      | Descripción |
|--------------------------|--------------|-------------|
| `From(in, out, amount)`  | **requerido**| Par + monto de entrada |
| `WithSlippage(v)`        | `0.5`        | Slippage en % |
| `WithMaxHops(n)`         | `2`          | Máximo de hops |
| `OnNetwork(n)`           | `NetworkBase`| Red objetivo |
| `WithPriceBase(s)`       | —            | Igual `.priceBase` |
| `WithDexs(...)`          | —            | Restringe DEXes |
| `WithBlockNumber(n)`     | `"latest"`   | Bloque específico |
| `WithRpcUrls(...)`       | RPC client   | Sobrescribe endpoints del quoter |

#### Quote

```typescript
interface Quote {
  tokenIn:           Address    // token de entrada
  tokenOut:          Address    // token de salida
  amountIn:          string     // monto de entrada legible
  amountOut:         string     // salida estimada legible
  minOut:            string     // salida mínima tras slippage (legible)
  amountInWei:       bigint     // entrada exacta — pasarlo a approve()
  amountOutWei:      bigint     // salida estimada (Wei)
  minOutWei:         bigint     // mínimo aplicado on-chain (Wei)
  steps:             Hex        // ruta codificada — pasada a AFI.swap()
  path:              Address[]  // direcciones de tokens en la ruta
  hops:              Hop[]      // detalle por hop
  slippage:          number     // slippage aplicado en %
  feeBps:            number     // tarifa del protocolo en la cotización
  tokenInPrice:      string     // precio de tokenIn en unidades de tokenOut
  tokenOutPrice:     string     // precio de tokenOut en unidades de tokenIn
  tokenInBasePrice?: string     // llenado por priceBase()
  tokenOutBasePrice?: string    // llenado por priceBase()
  createdAt:         number     // timestamp unix-ms — usado por isQuoteStale()
}

interface Hop {
  tokenIn:       Address
  tokenOut:      Address
  amountIn:      string
  amountOut:     string
  minOut:        string
  amountInWei:   bigint
  amountOutWei:  bigint
  minOutWei:     bigint
  tokenInPrice:  string
  tokenOutPrice: string
  slippage:      number
  type:          string    // protocolo de pool, p.ej. "v3", "v2"
  kind:          string    // motor de routing
  routeId:       number
  weight:        number
}
```

---

### Operaciones de escritura (requieren signer)

#### `connect(privateKey)` / `Connect(privateKey)`

Adjunta un signer. Acepta hex con o sin prefijo `0x`.

```typescript
client.connect("0x…")
const c = new AfiClient({ rpcUrl, privateKey: "0x…" })
```

```go
err := client.Connect("…")
```

`client.address()` (TS) / `client.Address()` (Go) devuelve la dirección
derivada de la clave, o la dirección cero cuando no hay signer.

#### `approve(token, amountWei)` / `Approve(ctx, token, amountWei)`

Envía un approve por el monto exacto para el router AFI. Devuelve un
`PendingTx` (hash disponible al instante) o **null** cuando el allowance
existente ya es suficiente — ahorrando una transacción.

Para tokens estilo USDT el SDK resetea el allowance a cero primero. Si el reset
en sí falla (y el approve siguiente también), ambos errores se muestran en la
`ApprovalError` resultante.

```typescript
const pending = await client.approve(quote.tokenIn, quote.amountInWei)
if (pending) {
  console.log("Tx de aprobación:", pending.txHash)
  await pending.wait()
}
```

```go
pending, err := client.Approve(ctx, quote.TokenIn, quote.AmountInWei)
if pending != nil {
    receipt, err := pending.Wait(ctx)
}
```

#### `simulate(quote)` / `Simulate(ctx, quote)`

Hace `eth_call` contra el router AFI. Resuelve (o devuelve `nil`) en éxito.
Lanza `SimulationFailedError` (TS) o devuelve `*AfiError{Code:"SIMULATION_FAILED"}`
(Go) con el motivo del revert si el swap revertiría. **Ninguna tx se envía en
cualquier caso.**

```typescript
try {
  await client.simulate(quote)
} catch (e) {
  if (isSimulationFailedError(e)) console.error("revertiría:", e.reason)
}
```

```go
if err := client.Simulate(ctx, quote); err != nil {
    log.Println("revertiría:", err)
}
```

#### `submitSwap(quote)` / `SubmitSwap(ctx, quote)`

Envía la tx de swap sin esperar confirmación. Devuelve un `PendingSwap` cuyo
`wait(opts?)` bloquea hasta confirmar.

#### `executeSwap(quote, opts?)` / `ExecuteSwap(ctx, quote, opts?)`

Corre la secuencia completa — saldo → approve → simulate → submit → wait.
Devuelve cuando el swap está confirmado.

```typescript
interface ExecuteOptions {
  confirmations?: number    // predet.: 1
  timeoutMs?:     number    // predet.: sin timeout
}
```

```go
type ExecuteOptions struct {
    Confirmations uint64
    TimeoutMs     int64
}
```

#### `swap(opts)` / `Swap(ctx, opts...)`

Conveniencia: cotiza y ejecuta. Usa el flujo por etapas o
`executeSwap(quote, opts)` cuando necesites confirmations/timeout o
confirmación manual entre cotización y ejecución.

#### `estimateSwapCost(quote)` / `EstimateSwapCost(ctx, quote)`

Proyecta el costo de gas sin enviar tx. Devuelve
[`SwapCostEstimate`](#swapcostestimate). Útil para mostrar "tarifa de red
estimada" antes de que el usuario firme.

#### Tipos de resultado

```typescript
interface PendingTx {
  txHash: Hex
  wait(opts?: WaitForTxOptions): Promise<TxReceipt>
}

interface PendingSwap {
  txHash: Hex
  wait(opts?: WaitForTxOptions): Promise<SwapResult>
}

interface SwapResult {
  txHash:      Hex
  blockNumber: bigint
  amountIn:    bigint     // amountIn real del evento SwapExecuted
  amountOut:   bigint     // amountOut real del evento SwapExecuted
  tokenIn:     Address
  tokenOut:    Address
  gasUsed:     bigint
}

interface TxReceipt {
  blockNumber: bigint
  gasUsed:     bigint
}

interface WaitForTxOptions {
  confirmations?: number   // predet.: 1
  timeoutMs?:     number   // predet.: sin timeout
}
```

---

### Utilidades de transacción

#### `waitForTx(hash, opts?)` / `WaitForTx(ctx, hash, opts?)`

Hace polling hasta que la tx alcance las confirmaciones deseadas. Útil para
hashes obtenidos fuera del SDK (persistidos en DB, viniendo de otro servicio,
en cola).

```typescript
const receipt = await client.waitForTx("0x…", { confirmations: 2, timeoutMs: 30_000 })
```

```go
receipt, err := client.WaitForTx(ctx, "0x…", afi.WaitForTxOptions{
    Confirmations: 2, TimeoutMs: 30_000, PollIntervalMs: 1_000,
})
```

#### `parseSwapResult(receipt)` / `ParseSwapResult(receipt)`

Decodifica el evento `SwapExecuted` desde cualquier receipt. Devuelve `null` /
`nil` cuando no hay log `SwapExecuted` (la tx no era un swap AFI).

```typescript
import { parseSwapResult } from "@afi-run/sdk"

const result = parseSwapResult(receipt) // SwapResult | null
```

```go
result, err := afi.ParseSwapResult(receipt) // nil cuando no hay SwapExecuted en el receipt
```

Úsalo en indexers, herramientas de replay, jobs en cola que guardan el hash
para reconciliar después, y tests end-to-end.

---

### Configuración

| Método | Descripción |
|---|---|
| `setApiUrl(url)` / `SetApiURL(url)` | Sobrescribe la URL base de la API AFI (predet. `https://rpc.afi.run`). |
| `setGasBufferPercent(n)` / `SetGasBufferPercent(n)` | Sobrescribe el buffer en runtime. `0` desactiva. |
| `setLogger(fn)` / `SetLogger(fn)` | Adjunta o reemplaza el logger de diagnóstico. |
| `clearTokensCache(network?)` / `ClearTokensCache(network?)` | Fuerza refetch en el próximo `getTokens()`. |

---

## Helpers

### Utilidades de dirección

```typescript
import {
  isAddress,
  checksumAddress,    // EIP-55
  isZeroAddress,
  equalAddresses,     // case-insensitive
  ZERO_ADDRESS,
} from "@afi-run/sdk"
```

```go
afi.IsAddress(s)          // exige prefijo "0x" (mismo comportamiento que viem/ethers)
afi.Checksum(s)           // string EIP-55
afi.IsZeroAddress(s)
afi.EqualAddresses(a, b)
afi.ZeroAddress           // common.Address{}
afi.ZeroAddressHex        // "0x00…00"
```

### Calculadora de slippage

```typescript
import { calculateMinOut, applySlippage } from "@afi-run/sdk"

const minOut = calculateMinOut(quote.amountOutWei, 0.5)  // 0.5% menos, redondeo hacia abajo
```

```go
minOut := afi.CalculateMinOut(quote.AmountOutWei, 0.5)
```

`slippagePct` está en porcentaje (`0.5` = 0,5 %). Negativos se acotan a 0.
Valores ≥ 100 devuelven 0.

### Conversión de unidades

```typescript
import { parseUnits, formatUnits } from "@afi-run/sdk"

parseUnits("1000", 6)              // 1_000_000_000n
formatUnits(1_000_000_000n, 6)     // "1000"
```

```go
wei, _ := afi.ParseUnits("1000", 6) // *big.Int
str   := afi.FormatUnits(wei, 6)    // "1000"
```

### URLs de explorer

```typescript
client.txUrl(result.txHash)             // https://basescan.org/tx/…
client.addressUrl(addr, NETWORK.BSC)    // https://bscscan.com/address/…

// Standalone:
import { txUrl, addressUrl, NETWORK_EXPLORERS } from "@afi-run/sdk"
txUrl(hash, NETWORK.BASE, "https://mi-explorer")   // base personalizado
```

```go
url, _ := client.TxURL(result.TxHash.Hex())
addr, _ := afi.AddressURL(walletAddr, afi.NetworkArbitrum)
```

Por defecto vienen en `NETWORK_EXPLORERS` / `afi.NetworkExplorers`,
sobrescribibles en runtime.

### Frescura de cotización

```typescript
import { isQuoteStale } from "@afi-run/sdk"

if (isQuoteStale(quote, 30)) {   // más vieja que 30 segundos
  quote = await client.quote(...).get()
}
```

```go
if quote.IsStale(30) {
    quote, _ = client.GetQuote(ctx, ...)
}
```

Las cotizaciones envejecen rápido (pocos segundos en pares volátiles). Siempre
recotiza antes del broadcast en flujos lentos (wallet hardware, multi-sig,
revisión manual).

### Decodificación de custom errors (`decodeRevertReason`, campos `decoded` en los errores)

El SDK trae los **9 custom errors del router AFI** pre-registrados (verificados
en Basescan), más los de OpenZeppelin (`Ownable*`, `ReentrancyGuardReentrantCall`)
y los built-in de Solidity (`Error(string)`, `Panic(uint256)`). Los reverts se
decodifican automáticamente; el resultado estructurado queda adjunto al error lanzado.

```typescript
try {
  await client.simulate(quote)
} catch (e) {
  if (isSimulationFailedError(e) && e.decoded) {
    // e.decoded = { name: "InsufficientFunds", signature: "InsufficientFunds(uint256)", args: [100n] }
    if (e.decoded.name === "InsufficientFunds") {
      toast.error(`El pool solo tiene ${e.decoded.args[0]} disponible`)
    }
  }
}
```

```go
err := client.Simulate(ctx, quote)
var afiErr *afi.AfiError
if errors.As(err, &afiErr) && afiErr.Decoded != nil {
    if afiErr.Decoded.Name == "InsufficientFunds" {
        log.Printf("el pool solo tiene %s disponible", afiErr.Decoded.Args[0])
    }
}
```

#### Errores decodificados que vienen en el SDK

| Error                                 | Origen        |
|---------------------------------------|---------------|
| `DifferentAssets(address,address)`    | router AFI    |
| `FeeTooHigh(uint16)`                  | router AFI    |
| `InsufficientFunds(uint256)`          | router AFI    |
| `InvalidRouteID(uint16)`              | router AFI    |
| `NotOperator()`                       | router AFI    |
| `OwnableInvalidOwner(address)`        | OpenZeppelin  |
| `OwnableUnauthorizedAccount(address)` | OpenZeppelin  |
| `ReentrancyGuardReentrantCall()`      | OpenZeppelin  |
| `ZeroAddress()`                       | router AFI    |
| `Error(string)`                       | Solidity      |
| `Panic(uint256)`                      | Solidity      |

#### Registrar los errores de tu propio contrato

```typescript
import { registerCustomErrors, decodeRevertReason } from "@afi-run/sdk"

registerCustomErrors([
  { type: "error", name: "MyContractError", inputs: [
    { name: "code", type: "uint256" },
    { name: "msg",  type: "string" },
  ]},
])

// Decodificar revert data crudo manualmente
const decoded = decodeRevertReason("0x…")
// o simplemente confiar en que vendrá adjunto a los próximos errores lanzados
```

```go
// Parse cualquier ABI con definiciones de error y regístrala globalmente.
a, _ := abi.JSON(strings.NewReader(`[{"type":"error","name":"MyContractError", ...}]`))
afi.RegisterCustomErrors(a)

decoded := afi.DecodeRevertReason(rawHexBytes) // *afi.DecodedRevert
```

---

### Tarifa de la transacción en los resultados

`SwapResult` y `TxReceipt` ahora exponen el costo real pagado:

```typescript
const result = await client.executeSwap(quote)
console.log(`Costo: ${result.feeEth} ETH (${result.feeWei} wei @ ${result.effectiveGasPrice} wei/gas)`)
```

Los mismos campos en Go:

```go
fmt.Printf("Costo: %s ETH\n", result.FeeETH)
```

Los receipts devueltos por `waitForTx`, `pending.wait()`, y `wait()` de
`approve`/`revoke` cargan la tarifa.

---

### `getTxStatus(hash)` — estado sin bloquear

Devuelve al instante el estado actual de una tx — útil para indicadores de UI
con polling donde bloquear en un receipt sería desastroso.

```typescript
const status = await client.getTxStatus(hash)
// "pending" | "success" | "failed" | "unknown"
```

```go
status, err := client.GetTxStatus(ctx, hash)
switch status {
case afi.TxStatusPending: // …
case afi.TxStatusSuccess: // …
}
```

---

### `getTokenPrice(in, out, opts?)` — consulta rápida de precio

Lookup liviano de tasa de cambio para un par sin comprometerse al swap.

```typescript
const { price, inverse } = await client.getTokenPrice(USDC, WETH)
// price   = "0.00031" (1 USDC en WETH)
// inverse = "3225"    (1 WETH en USDC)

// Sobrescribir defaults:
await client.getTokenPrice(USDC, WETH, { amount: "1000", slippage: 1.0, network: NETWORK.BSC })
```

```go
p, _ := client.GetTokenPrice(ctx, usdc, weth)
// p.Price, p.Inverse
```

---

### Gestión de nonce — `getNonce`, `useManagedNonce`

Para bots que envían varios swaps en paralelo sin esperar entre ellos.

```typescript
// Lectura única
const n = await client.getNonce()

// Modo gestionado (recomendado para bots)
await client.useManagedNonce()    // sincroniza desde la chain y mantiene un contador local
await Promise.all([
  client.executeSwap(quote1),
  client.executeSwap(quote2),
  client.executeSwap(quote3),     // cada uno toma un nonce único, sin race
])

// En caso de error / fork / replacement
await client.resetManagedNonce()  // re-sincroniza con la chain

// Override por llamada
await client.executeSwap(quote, { nonce: 142 })
```

```go
n, _ := client.GetNonce(ctx)
client.UseManagedNonce(ctx)
nonce := uint64(142)
client.ExecuteSwap(ctx, quote, afi.ExecuteOptions{Nonce: &nonce})
client.ResetManagedNonce(ctx)
client.DisableManagedNonce()
```

Cuando el contador local se desincroniza (tx rechazada, replacement), llama
`resetManagedNonce()` / `ResetManagedNonce(ctx)` para re-sincronizar.

---

### `preflight(quote)` — verificación combinada de listo-para-ejecutar

Corre balance + allowance + simulate **sin enviar tx** y devuelve un reporte
estructurado para que tu UI muestre "listo para hacer swap".

```typescript
const report = await client.preflight(quote)
if (!report.canExecute) {
  for (const p of report.problems) console.error(`${p.code}: ${p.message}`)
} else if (report.needsApproval) {
  showButton("Aprobar y Hacer Swap")
} else {
  showButton("Hacer Swap")
}
```

```go
report, _ := client.Preflight(ctx, quote)
if !report.CanExecute {
    for _, p := range report.Problems {
        fmt.Printf("%s: %s\n", p.Code, p.Message)
    }
}
```

`canExecute = problems.length === 0` — `needsApproval` es informativo porque
`executeSwap` se encarga del approve automáticamente.

---

### Transacciones pre-codificadas (`encodeSwap`, `encodeApprove`, `encodeRevoke`)

Para frontends donde la clave privada vive en una wallet del usuario (Wagmi,
RainbowKit, MetaMask, Frame, hardware wallet, Safe SDK), construye el calldata
con el SDK y envíalo a través del connector.

```typescript
import { encodeSwap, encodeApprove } from "@afi-run/sdk"

const approveTx = encodeApprove(quote.tokenIn, quote.amountInWei)
await walletClient.sendTransaction(approveTx)

const swapTx = encodeSwap(quote)
const hash = await walletClient.sendTransaction(swapTx)
```

```go
swapTx, _ := afi.EncodeSwap(quote)         // {To, Data, Value}
approveTx, _ := afi.EncodeApprove(usdc, amt)
revokeTx, _ := afi.EncodeRevoke(usdc)
```

Los tres también están expuestos como métodos del client (`client.encodeSwap(quote)`,
etc.) cuando ya tienes un client configurado.

---

### Revocar allowance — `revoke(token)` / `Revoke(ctx, token)`

Envía `approve(AFI, 0)` poniendo la allowance del router en cero. Devuelve
`null` / `nil` cuando la allowance ya es cero. Úsalo como limpieza de
seguridad post-swap.

```typescript
const tx = await client.revoke(quote.tokenIn)
await tx?.wait()
```

```go
pending, err := client.Revoke(ctx, quote.TokenIn)
if pending != nil {
    pending.Wait(ctx)
}
```

---

### Multicall genérico — `multicall(calls)` / `Multicall(ctx, calls)`

Empaqueta N lecturas arbitrarias en una sola llamada RPC vía Multicall3. Útil
para cualquier batch más allá de `tokenInfo` — precios de pools, tus propios
contratos, estado custom de DEX.

```typescript
import { ERC20_ABI } from "@afi-run/sdk"

const results = await client.multicall([
  { address: usdc, abi: ERC20_ABI, functionName: "totalSupply" },
  { address: weth, abi: ERC20_ABI, functionName: "totalSupply" },
])
for (const r of results) {
  if (r.status === "success") console.log(r.result)
}
```

```go
// El ABI de Multicall3 está en afi.Multicall3ABIJSON para uso de bajo nivel.
calls := []afi.Multicall3Call{ /* … */ }
results, err := client.Multicall(ctx, calls)
```

---

### Refrescar una cotización obsoleta — `refreshQuote(quote)` / `RefreshQuote(ctx, quote)`

Reobtiene la cotización con los parámetros originales (network, slippage,
maxHops, priceBase, dexs). Conveniencia para flujos lentos (confirmación en
hardware wallet, revisión multi-sig) donde el builder original ya no está.

```typescript
if (isQuoteStale(quote, 30)) {
  quote = await client.refreshQuote(quote)
}
```

```go
if quote.IsStale(30) {
    quote, _ = client.RefreshQuote(ctx, quote)
}
```

---

### Cache de metadata de tokens

`tokenInfo` / `tokenInfoBatch` mantienen `(symbol, name, decimals)` en un
cache en memoria — esos valores no cambian para un token ERC-20, así que el
segundo lookup de metadata cuesta **cero RPC**. Cuando hay `owner`, solo
balance/allowance se refrescan en llamadas posteriores.

Limpia el cache con `clearTokenMetadataCache()` (TS) /
`ClearTokenMetadataCache()` (Go) si cambias de proveedor RPC y quieres
re-verificar los tokens.

---

### Serialización JSON

`Quote`, `SwapResult` y `TokenInfo` traen campos `bigint` / `*big.Int` que
rompen `JSON.stringify` y pierden precisión en `json.Marshal`. El SDK provee
helpers de ida y vuelta (bigints como strings base-10).

```typescript
import {
  bigintReplacer,
  quoteToJSON,    quoteFromJSON,
  swapResultToJSON, swapResultFromJSON,
  tokenInfoToJSON,  tokenInfoFromJSON,
} from "@afi-run/sdk"

// Guardar
await db.put(`quote:${id}`, JSON.stringify(quoteToJSON(quote)))
// Alternativa genérica para objetos arbitrarios:
JSON.stringify(cualquierObjeto, bigintReplacer)

// Restaurar
const restored = quoteFromJSON(await db.get(`quote:${id}`))
```

```go
// Quote / SwapResult / TokenInfo implementan MarshalJSON y UnmarshalJSON.
data, _ := json.Marshal(quote)        // bigints como strings
var q afi.Quote
_ = json.Unmarshal(data, &q)
```

### ABIs exportadas

```typescript
import { AFI_ABI, ERC20_ABI, MULTICALL3_ABI } from "@afi-run/sdk"
// drop-in para viem.readContract / writeContract / parseEventLogs
```

```go
// Strings JSON crudas — pasa a abi.JSON(strings.NewReader(...)).
afi.AFIABIJSON
afi.ERC20ABIJSON
afi.Multicall3ABIJSON
```

---

## Logs y diagnóstico

Adjunta un logger para capturar tiempo y resultado de las operaciones
principales.

```typescript
new AfiClient({
  rpcUrl,
  logger: (e) => console.log(`${e.method} ${e.durationMs}ms ok=${e.ok}`),
})

interface LogEvent {
  kind:        "rpc" | "api"
  method:      string        // "getQuote", "approve", "simulate", "submitSwap", "executeSwap"
  durationMs:  number
  ok:          boolean
  error?:      unknown
}
```

```go
afi.NewClient(afi.Config{
    RPCURL: url,
    Logger: func(e afi.LogEvent) {
        log.Printf("%s %dms ok=%v", e.Method, e.DurationMs, e.OK)
    },
})
```

Cambia en runtime con `setLogger(fn)` / `SetLogger(fn)`. Pasa `undefined` /
`nil` para desactivar.

`health()` / `Health(ctx)` sondea RPC (chain ID) y API AFI en paralelo:

```typescript
const h = await client.health()
if (!h.rpc.ok || !h.api.ok) {
  console.error("no listo:", h)
  process.exit(1)
}
```

---

## Manejo de errores

Todos los errores lanzados derivan de `AfiError` (TS) / `*AfiError` (Go). El
`Code` identifica la clase; el `Message` es amigable; algunos códigos adjuntan
campos extra.

### Referencia de códigos de error

| Código | Cuándo | Campos extra |
|---|---|---|
| `NO_SIGNER` | Método de escritura llamado sin `connect()`. | — |
| `INSUFFICIENT_BALANCE` | Saldo de tokenIn por debajo del requerido. | `token`, `owner`, `symbol`, `decimals`, `balance`, `required` |
| `APPROVAL_FAILED` | `approve()` (o el reset estilo USDT) revirtió. | — |
| `SIMULATION_FAILED` | `eth_call` de `AFI.swap(...)` revirtió antes de enviar tx. | `reason`, `revertData?` (TS) |
| `QUOTE_FAILED` | La API del quoter devolvió error (sin ruta, validación, …). | — |
| `SWAP_REVERTED` | La tx de swap revirtió on-chain, o `estimateGas` falló. | `reason` (TS) |

Cuando ocurre `INSUFFICIENT_BALANCE`, el SDK hace **un multicall extra** para
adjuntar `symbol` y `decimals` — el mensaje sale como
*"Insufficient USDC for 0xABcd…: have 0.5, need 1"* en vez de direcciones crudas.

### TypeScript — type guards

Prefiere los guards a `instanceof` — sobreviven a class shims y a la
transpilación de `esbuild`, `swc`, interop ESM↔CJS, etc.

```typescript
import {
  isAfiError,
  isInsufficientBalanceError,
  isQuoteError,
  isSimulationFailedError,
  isApprovalError,
  isSwapRevertedError,
  isNoSignerError,
} from "@afi-run/sdk"

try {
  await client.executeSwap(quote)
} catch (e) {
  if (isInsufficientBalanceError(e)) {
    showFundingPrompt(e.symbol ?? e.token, e.required - e.balance)
  } else if (isSimulationFailedError(e)) {
    toast.error(`Revertiría: ${e.reason}`)
  } else if (isQuoteError(e)) {
    toast.error("Sin ruta disponible.")
  } else {
    Sentry.captureException(e)
    throw e
  }
}
```

### Go

```go
result, err := client.ExecuteSwap(ctx, quote)
if err != nil {
    var afiErr *afi.AfiError
    if errors.As(err, &afiErr) {
        switch afiErr.Code {
        case "NO_SIGNER":
            log.Println("conecta un signer primero")
        case "INSUFFICIENT_BALANCE":
            log.Printf("faltan %s %s", new(big.Int).Sub(afiErr.Required, afiErr.Balance), afiErr.Symbol)
        case "SIMULATION_FAILED":
            log.Println("revertiría:", afiErr.Message)
        case "QUOTE_FAILED":
            log.Println("sin ruta")
        case "APPROVAL_FAILED":
            log.Println("approve revirtió")
        case "SWAP_REVERTED":
            log.Println("revert on-chain")
        }
        return
    }
    log.Fatal(err) // error de red / encoding / programación
}
```

---

## Modelo de seguridad

| Riesgo                            | Mitigación |
|-----------------------------------|------------|
| Bypass de slippage                | `minOutWei` siempre viene del quoter; los valores cero se rechazan en el cliente. |
| Aprobación excesiva               | Aprueba siempre el `amountInWei` exacto — nunca `MAX_UINT256`. |
| Falla estilo USDT                 | El SDK resetea a 0 antes de re-aprobar; los errores del reset se preservan y se muestran si el approve siguiente también falla. |
| Tx que revertiría                 | `simulate` corre antes de cada `executeSwap` — las fallas lanzan sin gastar gas. |
| Race en allowance                 | Se vuelve a leer la allowance on-chain tras cada approve. |
| Subestimación de gas              | `eth_estimateGas` × `(1 + gasBufferPercent/100)` (predet. +15 %). |
| Mismatch de red                   | El signer lee el chain ID del RPC (Go); el SDK expone `chainId()` para verificarlo (TS). |
| Cotizaciones obsoletas            | `Quote.createdAt` se setea en la captura. Usa `isQuoteStale(quote, maxAge)` antes de enviar. |
| ETH nativo pasado por error       | El router no acepta ETH nativo; usa WETH. |

La dirección del router y la tarifa del protocolo se leen on-chain en cada
cotización — el SDK no confía en valores cacheados aquí.

---

## Recetas

### Guardar una cotización y restaurarla después

```typescript
import { quoteToJSON, quoteFromJSON, isQuoteStale } from "@afi-run/sdk"

await redis.set(`quote:${userId}`, JSON.stringify(quoteToJSON(quote)))
// …
const raw = await redis.get(`quote:${userId}`)
const quote = quoteFromJSON(raw!)
if (isQuoteStale(quote, 60)) {
  // recotizar
}
```

### Saltar approve cuando ya hay allowance suficiente

```typescript
if (await client.hasAllowance(quote.tokenIn, quote.amountInWei)) {
  // sin approve
} else {
  const tx = await client.approve(quote.tokenIn, quote.amountInWei)
  await tx?.wait()
}
```

```go
ok, _ := client.HasAllowance(ctx, quote.TokenIn, quote.AmountInWei)
if !ok {
    pending, _ := client.Approve(ctx, quote.TokenIn, quote.AmountInWei)
    if pending != nil {
        _, _ = pending.Wait(ctx)
    }
}
```

### Mostrar tarifa estimada antes de firmar

```typescript
const cost = await client.estimateSwapCost(quote)
toast.info(`Tarifa de red estimada: ~${cost.totalEth} ETH`)
```

### Dashboard de portafolio — info en lote para N tokens

```typescript
const tokens = await client.getTokens()
const infos = await client.tokenInfoBatch(
  tokens.filter(t => t.active).map(t => t.address),
  "self",
)
infos.forEach(i => console.log(`${i.symbol}: ${formatUnits(i.balance ?? 0n, i.decimals)}`))
```

### Fail-fast en el arranque

```typescript
const h = await client.health()
if (!h.rpc.ok || !h.api.ok) {
  console.error("SDK AFI no listo", h)
  process.exit(1)
}
const net = await client.detectNetwork()
if (net !== "base") {
  console.error(`Esperaba RPC para base, llegó ${net}`)
  process.exit(1)
}
```

### Bot que espera 2 confirmaciones

```typescript
const result = await client
  .quote(USDC, WETH, "500")
  .slippage(1.0)
  .execute({ confirmations: 2, timeoutMs: 60_000 })
```

```go
result, err := client.ExecuteSwap(ctx, quote, afi.ExecuteOptions{
    Confirmations: 2, TimeoutMs: 60_000,
})
```

### Reprocesar o indexar una tx conocida

```typescript
import { parseSwapResult } from "@afi-run/sdk"
const receipt = await publicClient.getTransactionReceipt({ hash })
const result = parseSwapResult(receipt)
if (result) await indexSwap(result)
```

---

## Redes y constantes

### Redes soportadas

| Red         | Chain ID | Explorer predet.             | Constante TS           | Constante Go           |
|-------------|----------|------------------------------|------------------------|------------------------|
| Base        | 8453     | https://basescan.org         | `NETWORK.BASE`         | `afi.NetworkBase`      |
| BSC         | 56       | https://bscscan.com          | `NETWORK.BSC`          | `afi.NetworkBSC`       |
| Arbitrum    | 42161    | https://arbiscan.io          | `NETWORK.ARBITRUM`     | `afi.NetworkArbitrum`  |
| Ethereum    | 1        | https://etherscan.io         | `NETWORK.ETHEREUM`     | `afi.NetworkEthereum`  |
| Unichain    | 130      | https://uniscan.xyz          | `NETWORK.UNICHAIN`     | `afi.NetworkUnichain`  |

Sobrescribe explorers en runtime vía `NETWORK_EXPLORERS` (TS) o
`afi.NetworkExplorers` (Go).

### DEXes soportadas

```typescript
import { DEX } from "@afi-run/sdk"
// DEX.UNI_V3 · DEX.UNI_V4 · DEX.CAKE_V3 · DEX.AERODROME
// DEX.BALANCER · DEX.CURVE128 · DEX.CURVE256 · DEX.FLUID
```

```go
// afi.DexUniV3 · afi.DexUniV4 · afi.DexCakeV3 · afi.DexAerodrome
// afi.DexBalancer · afi.DexCurve128 · afi.DexCurve256 · afi.DexFluid
```

### Contratos desplegados por chain

Despliegue el 2026-05-30. Todos los contratos verificados en el respectivo block explorer.

**Afi (router de swap del usuario)** — `AFI_ADDRESSES` (TS) / `afi.AfiAddresses` (Go):

| Chain | Dirección |
|---|---|
| Ethereum (1) | `0xc578a4e89795803F396160610F4990c44abA8dAb` |
| BSC (56) | `0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e` |
| Unichain (130) | `0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e` |
| Base (8453) | `0xFd4F8822f13D01aB142Bc985Ce587E35d7673C6e` |
| Arbitrum (42161) | `0xd74F60BD38243d089e286E3B6b9348f43a2314dF` |

**RouteQuoter (simulación off-chain vía `eth_call` + `state_override`)** — `ROUTE_QUOTER_ADDRESSES` (TS) / `afi.RouteQuoterAddresses` (Go):

| Chain | Dirección |
|---|---|
| Ethereum | `0x5e41b417E9742DB9c5402F8B1969a33891628Bed` |
| BSC | `0xcA37E05a20E93fD88E5367F9d7d1422937c57A38` |
| Unichain | `0x2Cc852Cd57CC1b57CA09dbA7f69F0e225008cEBE` |
| Base | `0xB5637138Cee6e757B679FFF8aDEA8DBa3E7544bB` |
| Arbitrum | `0xBdD42B4fF06aCa8908D5E5d4826fFf5cdaC43895` |

**NMR (NathanMayerRothschild — arbitraje vía flash loan)** — `NMR_ADDRESSES` (TS) / `afi.NMRAddresses` (Go). Solo chains con Aave V3:

| Chain | Dirección |
|---|---|
| Ethereum | `0x29EfbFC1534A9B7af02142A5D97454E24Dc51b3a` |
| Base | `0xefA12ba0196FD5ec44AF2ecAddc17333dF5FA779` |
| Arbitrum | `0x6b533D53ec93eC30963b38576Ed8330Ff346a723` |

### Otras constantes

| Nombre                  | Valor |
|-------------------------|-------|
| Multicall3 (todas)      | `0xcA11bde05977b3631167028862bE2a173976CA11` |
| Permit2 (todas)         | `0x000000000022D473030F116dDEE9F6B43aC78BA3` |
| URL base de la API      | `https://rpc.afi.run` |
| Buffer de gas predet.   | `15` (%) |

```typescript
import {
  AFI_ADDRESSES,           // Record<chainId, Address>
  ROUTE_QUOTER_ADDRESSES,  // Record<chainId, Address>
  NMR_ADDRESSES,           // Record<chainId, Address> (Eth, Base, Arb)
  AFI_ABI,
  NMR_ABI,
  MULTICALL3_ADDRESS,
  DEFAULT_GAS_BUFFER_PERCENT,
  NETWORK_CHAIN_IDS,
} from "@afi-run/sdk"
```

```go
afi.AfiAddresses           // map[Network]common.Address
afi.RouteQuoterAddresses   // map[Network]common.Address
afi.NMRAddresses           // map[Network]common.Address (Eth, Base, Arb)
afi.AfiABI                 // string (JSON)
afi.NMRABI                 // string (JSON)
afi.Multicall3Address      // common.Address
afi.DefaultGasBufferPercent
afi.NetworkChainIDs        // map[Network]int64
```

### Helpers de bajo nivel

Para construir flujos de operator o arbitraje directamente (sin pasar por
`client.swap()` de alto nivel).

**Encoder de steps en formato compacto** — produce los bytes que consume `Lib.runRoutes`:

```typescript
import { encodeSteps, type Step } from "@afi-run/sdk"

const steps: Step[] = [
  { id: 3, data: "0x...59-byte-stepData..." }, // UniV3 route id=3
]
const params = encodeSteps(steps) // -> Hex
// Usa `params` como último arg de Afi.swap / Afi.swapFor / NMR.requestOperation
```

```go
import afi "github.com/afi-run/sdk/go"

params, err := afi.EncodeSteps([]afi.Step{
    {ID: 3, Data: stepData}, // UniV3
})
```

Layout: `uint8 numSteps + [uint16 id | uint16 dataLen | bytes data] × N`.

**Encoders de transacciones NMR** — builders de calldata para acciones de operator y owner:

```typescript
import {
  encodeNMRRequestOperation,
  encodeNMRSwap,
  encodeNMRLoan,
  encodeNMRSweepProfit,
  encodeNMRSetTreasury,
} from "@afi-run/sdk"
```

```go
// El SDK Go expone la misma API:
// afi.EncodeNMRRequestOperation, afi.EncodeNMRSwap, afi.EncodeNMRLoan,
// afi.EncodeNMRSweepProfit, afi.EncodeNMRSetTreasury
```

| Función | Quién llama | Propósito |
|---|---|---|
| `requestOperation(asset, amount, params)` | operator | Dispara flash loan de Aave; el callback ejecuta la cadena de rutas |
| `swap(asset, amount, minOut, params)` | operator | Ciclo de arbitraje — token de entrada == token de salida |
| `loan(user, asset, amount, minOut, params)` | operator | Toma del usuario, ejecuta la ruta, devuelve la porción del usuario |
| `sweepProfit(asset, amount)` | operator | Retira la ganancia acumulada del NMR hacia el `treasury` |
| `setTreasury(addr)` | owner | Actualiza el destino de las ganancias |

**Cuándo usar `swap` / `swapFor` / `batchSwapFor`**

El router Afi expone tres puntos de entrada de ejecución. Elige según quién paga el gas y de dónde salen los tokens de entrada:

| Función | Quién llama | Caso de uso |
|---|---|---|
| `Afi.swap(tokenIn, amount, tokenOut, minOut, params)` | usuario final (el propio `msg.sender` paga) | Usuario de DApp intercambiando sus propios tokens |
| `Afi.swapFor(user, tokenIn, amount, tokenOut, minOut, params)` | operator (el operator toma de `user`) | Bot ejecutando en nombre de un usuario pre-aprobado |
| `Afi.batchSwapFor(SwapRequest[])` | operator | Varios usuarios en una sola tx — batch eficiente en gas |

Importante: `swapFor` y `batchSwapFor` requieren que `user` haya llamado antes a `IERC20(tokenIn).approve(Afi, amount)`. Sin esa allowance el `transferFrom` del operator revierte.

**NMR — `swap` vs `loan` vs `requestOperation`**

El contrato NMR es solo-operator, pero ofrece cuatro primitivas distintas. Elige por intención, no por nombre — `swap` aquí **no** es un intercambio direccional:

| Función | Quién llama | Característica distintiva |
|---|---|---|
| `NMR.requestOperation(asset, amount, params)` | operator | Dispara flash loan en Aave; la ganancia del ciclo se acumula en NMR |
| `NMR.swap(asset, amount, minOut, params)` | operator | **Ciclo de arbitraje — requiere `tokenIn == tokenOut`** (el ciclo termina donde empezó). Revierte en caso contrario. |
| `NMR.loan(user, asset, amount, minOut, params)` | operator | Toma `amount` de `user` (requiere `user.approve(NMR)`); ejecuta la ruta; devuelve `amount + userShare(ganancia)` al usuario; el resto queda en NMR. |
| `NMR.sweepProfit(asset, amount)` | operator | Retira la ganancia acumulada en NMR hacia el `treasury` configurado. |

Atención: **`NMR.swap` REQUIERE `tokenIn == tokenOut`** — llamarla con tokens distintos revierte con `OutputAssetMismatch`. Para un intercambio direccional en nombre de un usuario, usa `NMR.loan` (o `Afi.swapFor` en el router de usuario).

**Encoders de admin / owner**

El SDK ahora expone los encoders solo-owner del Afi para dashboards y flujos de gobernanza. Son solo builders de calldata — el broadcast queda a cargo del que llama:

```typescript
import {
  encodeAfiPause, encodeAfiUnpause,
  encodeAfiSetTreasury, encodeAfiSetFeeBps,
  encodeAfiSetUserFeeBps, encodeAfiSetUserFeeBpsBatch,
  encodeAfiClearUserFeeBps, encodeAfiResetAnyUserOverride,
  encodeAfiAddRule, encodeAfiClearRules,
  encodeAfiSetOperator, encodeAfiRescueTokens,
} from "@afi-run/sdk"
```

```go
// afi.EncodeAfiPause / EncodeAfiUnpause / EncodeAfiSetTreasury / EncodeAfiSetFeeBps /
// EncodeAfiSetUserFeeBps / EncodeAfiSetUserFeeBpsBatch / EncodeAfiClearUserFeeBps /
// EncodeAfiResetAnyUserOverride / EncodeAfiAddRule / EncodeAfiClearRules /
// EncodeAfiSetOperator / EncodeAfiRescueTokens
```

Quien firme estas txs debe ser el owner del Afi. Los cambios de comisión están limitados on-chain por `MAX_FEE_BPS = 50` (0,50 %) — cualquier valor superior revierte con `FeeTooHigh`.

**Parsers de eventos**

Para indexers, dashboards y actualizaciones de UI post-tx, el SDK incluye parsers tipados para cada evento que emiten Afi y NMR. Cada helper recibe los `logs` del receipt y retorna un array de eventos decodificados (vacío cuando no hay log correspondiente):

```typescript
import {
  parseSwapExecuted, parseFeeCollected, parseTreasuryUpdated,
  parseFeeBpsUpdated, parseUserFeeBpsSet, parseUserFeeBpsCleared,
  parseFlashLoanRequested, parseFlashLoanExecuted, parseFlashLoanFailed,
  parseProfitSwept, parseProfitShareUpdated,
} from "@afi-run/sdk"

// Uso:
const events = parseSwapExecuted(receipt.logs)
// events: [{ from, assetIn, amountIn, assetOut, amountOut }, ...]
```

```go
// Equivalentes en el SDK Go: afi.ParseSwapExecuted, afi.ParseFeeCollected,
// afi.ParseTreasuryUpdated, afi.ParseFeeBpsUpdated, afi.ParseUserFeeBpsSet,
// afi.ParseUserFeeBpsCleared, afi.ParseFlashLoanRequested,
// afi.ParseFlashLoanExecuted, afi.ParseFlashLoanFailed,
// afi.ParseProfitSwept, afi.ParseProfitShareUpdated
```

Úsalos para alimentar libros contables (`FeeCollected`, `ProfitSwept`), dashboards de gobernanza (`TreasuryUpdated`, `FeeBpsUpdated`, `ProfitShareUpdated`), overrides de comisión por usuario (`UserFeeBpsSet` / `UserFeeBpsCleared`) y telemetría de flash loans (`FlashLoanRequested` / `Executed` / `Failed`).

---

## Flujos de operador

Snippets de extremo a extremo para las cuatro superficies de operador del
protocolo. Todos siguen el mismo patrón: arma la ruta vía el quoter (o
`encodeSteps`), firma con la clave del operador, broadcast.

### Swap en nombre de 1 usuario — `swapFor`

`Afi.swapFor` permite que un operador ejecute una cotización donde los tokens
de entrada salen de `user` (que debe haber hecho `approve(Afi, amount)` antes)
y la salida vuelve al mismo `user`. El operador paga el gas — el usuario no
necesita gastar ETH.

```typescript
import { AfiClient, NETWORK } from "@afi-run/sdk"

const client = new AfiClient({ rpcUrl: process.env.RPC_URL!, privateKey: process.env.OP_KEY! as `0x${string}` })

const result = await client.swapFor({
  user:     "0xUser...",
  tokenIn:  USDC,
  tokenOut: WETH,
  amountIn: "1000",
  slippage: 0.5,
  network:  NETWORK.BASE,
})
console.log("Swap hecho para", result.user, "tx:", client.txUrl(result.txHash))
```

### Batch — `batchSwapFor`

Ejecuta varias cotizaciones en una sola transacción. Más eficiente en gas
cuando tienes múltiples usuarios pre-aprobados que liquidar en el mismo
bloque.

```typescript
const results = await client.batchSwapFor([
  { user: "0xUserA...", tokenIn: USDC, tokenOut: WETH, amountIn: "500" },
  { user: "0xUserB...", tokenIn: USDC, tokenOut: WETH, amountIn: "750" },
  { user: "0xUserC...", tokenIn: DAI,  tokenOut: WETH, amountIn: "1000" },
], { slippage: 0.5 })

for (const r of results) console.log(r.user, "->", r.amountOut)
```

### Ciclo de arbitraje con flash loan — `executeNMRArbitrage`

Dispara `NMR.requestOperation`, que pide `amount` de `asset` prestado a Aave,
ejecuta la ruta codificada en `params`, paga el préstamo + premium, y acumula
el delta en NMR. La ruta debe terminar en `asset` (ciclo).

```typescript
import { encodeSteps } from "@afi-run/sdk"

const params = encodeSteps([
  { id: 3, data: stepUniV3 },     // USDC -> WETH en Uniswap V3
  { id: 7, data: stepCurve },     // WETH -> USDC en Curve
])

const result = await client.executeNMRArbitrage({
  asset:  USDC,
  amount: 100_000n * 10n ** 6n,   // 100k USDC vía flash loan
  params,
})
console.log("ganancia:", result.profitWei, "wei")
```

### Arbitraje financiado por el usuario — `nmrLoanArbitrage` (NMR.loan)

Toma `amount` desde `user` (sin flash loan), corre el ciclo, devuelve
`amount + userShare(profit)` al `user`. El NMR retiene la parte del operador.

```typescript
const result = await client.nmrLoanArbitrage({
  user:   "0xUser...",
  asset:  USDC,
  amount: "5000",
  minOut: "5000",                // piso para la salida del ciclo
  params,
})
console.log("devuelto al usuario:", result.userAmountOut, "parte del operador:", result.operatorShare)
```

### Retirar ganancia del NMR — `sweepNMRProfit`

Mueve la ganancia acumulada del NMR a la `treasury` configurada. Sólo
operador.

```typescript
await client.sweepNMRProfit({ asset: USDC, amount: 10_000n * 10n ** 6n })
```

---

## Admin / gobernanza

El router Afi es `Ownable` — estos flujos requieren la **clave de owner** (no
sólo operador). Lee primero el estado on-chain, luego arma y envía la tx.

### Inspeccionar el estado actual

```typescript
const paused = await client.isPaused()
const feeBps = await client.getFeeBps()
const userFee = await client.getUserFeeBps("0xUser...")  // 0 si no hay override
console.log({ paused, globalFeeBps: feeBps, userOverrideBps: userFee })
```

### Pause / unpause

```typescript
const tx = await client.adminPause()        // bloquea swaps nuevos; tx pendientes terminan
await tx.wait()
// ...
await (await client.adminUnpause()).wait()
```

### Cambiar la tarifa del protocolo

Limitada on-chain por `MAX_FEE_BPS = 50` (0.50%). Valores por encima hacen
revert con `FeeTooHigh`.

```typescript
await (await client.adminSetFeeBps(25)).wait()                 // 0.25% global
await (await client.adminSetUserFeeBps("0xVIP...", 5)).wait()  // 0.05% para un único usuario
await (await client.adminClearUserFeeBps("0xVIP...")).wait()   // quitar el override
```

### Agregar una regla de validación

Las reglas son contratos externos que implementan `IAfiRule`. El router llama
a cada regla registrada antes de cada swap; cualquier revert aborta la
operación.

```typescript
await (await client.adminAddRule("0xRule...")).wait()
await (await client.adminClearRules()).wait()
```

La clave que firma cualquier `admin*` debe coincidir con el owner del Afi —
de lo contrario se lanza `OwnableUnauthorizedAccount`. Lo mismo aplica a
`adminSetTreasury`, `adminSetOperator(addr, true|false)` y
`adminRescueTokens(token, amount, to)`.

---

## Indexación de eventos

Cada evento emitido por Afi y NMR tiene un parser tipado que recibe un array
de `logs` y devuelve las entradas decodificadas. Combínalo con `getLogs` para
construir indexers, dashboards y ledgers post-tx.

```typescript
import {
  parseSwapExecuted, parseFeeCollected,
  parseFlashLoanRequested, parseFlashLoanExecuted, parseFlashLoanFailed,
  parseProfitSwept, parseProfitShareUpdated,
  parseTreasuryUpdated, parseFeeBpsUpdated,
  parseUserFeeBpsSet, parseUserFeeBpsCleared,
  AFI_ADDRESSES, NMR_ADDRESSES,
} from "@afi-run/sdk"
import { createPublicClient, http } from "viem"
import { base } from "viem/chains"

const publicClient = createPublicClient({ chain: base, transport: http(process.env.RPC_URL!) })

// 1. Trae los logs del rango deseado
const logs = await publicClient.getLogs({
  address: [AFI_ADDRESSES[8453], NMR_ADDRESSES[8453]],
  fromBlock: 22_000_000n,
  toBlock:   22_005_000n,
})

// 2. Decodifica por tipo de evento
const swaps         = parseSwapExecuted(logs)
const fees          = parseFeeCollected(logs)
const flashRequests = parseFlashLoanRequested(logs)
const flashExecuted = parseFlashLoanExecuted(logs)
const flashFailed   = parseFlashLoanFailed(logs)
const profitSwept   = parseProfitSwept(logs)
const profitShare   = parseProfitShareUpdated(logs)

for (const s of swaps) console.log(`${s.from} ${s.amountIn} ${s.assetIn} -> ${s.amountOut} ${s.assetOut}`)
```

Cada parser devuelve `[]` cuando no hay logs compatibles, así que puedes
encadenarlos sin miedo. Los bigints llegan como `bigint` nativo. Los 11
parsers cubren:

| Parser | Origen | Uso |
|---|---|---|
| `parseSwapExecuted` | Afi | Liquidación por swap |
| `parseFeeCollected` | Afi | Ledger de fee del protocolo |
| `parseTreasuryUpdated` | Afi/NMR | Auditoría de gobernanza |
| `parseFeeBpsUpdated` | Afi | Cambio de fee global |
| `parseUserFeeBpsSet` | Afi | Override por usuario creado |
| `parseUserFeeBpsCleared` | Afi | Override por usuario removido |
| `parseFlashLoanRequested` | NMR | Préstamo iniciado |
| `parseFlashLoanExecuted` | NMR | Ganancia del ciclo |
| `parseFlashLoanFailed` | NMR | Ciclo revertido |
| `parseProfitSwept` | NMR | Entrada al treasury |
| `parseProfitShareUpdated` | NMR | Gobernanza del share del operador |

---

## Builders de step por DEX

Para operadores que quieran saltarse el quoter HTTP y armar sus propias rutas
(estrategias MEV custom, backtests totalmente on-chain, tests de integración),
el SDK expone un builder por DEX soportado. Cada uno devuelve el `stepData` de
59 bytes más el route ID esperado por `Lib.runRoutes`.

```typescript
import {
  buildUniV3Step, buildCakeV3Step, buildUniV4Step, buildAerodromeStep,
  buildBalancerV3Step, buildFluidStep, buildCurve128Step, buildCurve256Step,
  buildAaveLiquidatorStep,
  encodeSteps,
} from "@afi-run/sdk"
```

| Builder | Campos requeridos |
|---|---|
| `buildUniV3Step({ tokenOut, fee, minOut, sqrtPriceLimitX96 })` | Pools Uniswap V3 (fee tier `500/3000/10000`) |
| `buildCakeV3Step({ tokenOut, fee, minOut, sqrtPriceLimitX96 })` | PancakeSwap V3 (mismo formato que Uni V3) |
| `buildUniV4Step({ currency0, currency1, fee, tickSpacing, hooks, zeroForOne, minOut })` | PoolKey + dirección Uniswap V4 |
| `buildAerodromeStep({ pool, tokenOut, tickSpacing, minOut })` | Aerodrome Slipstream — atención: `tickSpacing` es **int24 con signo** |
| `buildBalancerV3Step({ pool, tokenOut, minOut })` | Balancer V3 (one-hop, exactIn) |
| `buildFluidStep({ pool, swap0to1, tokenOut, minOut })` | Pools de Fluid DEX |
| `buildCurve128Step({ i, j, minDy, pool, tokenOut })` | Curve plain pools con índices int128 |
| `buildCurve256Step({ i, j, minDy, pool, tokenOut })` | Curve cryptoswap / meta con índices uint256 |
| `buildAaveLiquidatorStep({ pool, user, collateralAsset })` | Liquidación Aave V3 (ruta especial) |

### Ejemplo — ruta multi-hop alimentada directo a `client.swap()`

```typescript
const stepA = buildUniV3Step({
  tokenOut: WETH,
  fee: 500,
  minOut: 0n,                 // hop intermedio — sin piso
  sqrtPriceLimitX96: 0n,
})

const stepB = buildCurve128Step({
  i: 2, j: 0,
  minDy: minOutFinalWei,      // piso en el hop final
  pool: "0xCurvePool...",
  tokenOut: USDC,
})

const params = encodeSteps([
  { id: 3, data: stepA.data },   // 3 = UniV3
  { id: 6, data: stepB.data },   // 6 = Curve128
])

const tx = await client.swap({
  tokenIn:    USDC,
  tokenOut:   USDC,             // ciclo
  amountInWei,
  minOutWei:  minOutFinalWei,
  params,
})
```

Usa esta superficie cuando el quoter HTTP no esté disponible, cuando quieras
rutas determinísticas para tests de replay, o cuando tu estrategia necesite
parámetros de pool que el quoter no expone.

---

## Endpoints HTTP del quoter

El servicio afi-rpc ofrece varios endpoints más allá de `/quoter`. El client
envuelve cada uno con inputs y outputs tipados, así que los enchufas en código
TS/Go sin escribir boilerplate de fetch.

| Método | Endpoint | Devuelve | Para qué sirve |
|---|---|---|---|
| `client.findArbitrage(req)` | `POST /arbitrage` | `RouteQuote[]` | Rutas candidatas para un ciclo (usa `tokenIn === tokenOut`) |
| `client.findPath(req)` | `POST /command {action:"path"}` | `PathQuote` | Ruta multi-hop con precio para un camino explícito |
| `client.getRoutes(req)` | `POST /command {action:"routes"}` | `Route[]` | Caminos de token candidatos para un par |
| `client.priceQuote(req)` | `POST /command {action:"price"}` | `RouteQuote[]` | Cotizaciones por DEX para un par |
| `client.quoteDex(dex, req)` | `POST /command {action:<dex>}` | `RouteQuote[]` | Cotizaciones de un único DEX |
| `client.getLiquidationCandidates(req)` | `POST /aave` | `AavePosition[]` | Posiciones Aave V3 elegibles para liquidar |
| `client.liquidate(req)` | `POST /liquidation-call` | `LiquidationResult` | Ruta repay+swap para un liquidationCall |

`findArbitrage`, `priceQuote` y `quoteDex` devuelven `RouteQuote[]` — una lista de
rutas single-DEX ejecutables. Elige la mejor (`routeProfit(r)` =
`amountOutRaw − amountInRaw`) y arma un `Quote` ejecutable con `quoteFromRoute`,
que envuelve el hop `{routeId, stepData}` en los params de `Afi.swap` vía `encodeSteps`:

```typescript
import { quoteFromRoute, routeProfit } from "@afi-run/sdk"

// Ciclo self-funded: tokenIn === tokenOut, sin necesidad de operador.
const routes = await client.findArbitrage({ network: "base", tokenIn: USDC, tokenOut: USDC, amountIn: "1000" })
const best = routes.reduce((a, b) => (routeProfit(b)! > routeProfit(a)! ? b : a))

// Piso de salida en el principal — un ciclo no rentable revierte on-chain.
const quote = quoteFromRoute(best, BigInt(best.amountInRaw))
await client.executeSwap(quote)

const candidates = await client.getLiquidationCandidates({ network: "base" })
if (candidates.length > 0) {
  console.log(candidates[0].user, candidates[0].debtAmount, candidates[0].collaterals)
}
```

Todas las peticiones respetan el mismo override de `rpcUrls` (por llamada) y
levantan `QuoteError` ante fallos de validación — envuélvelas con
`isQuoteError(e)` para mensajes limpios en la UX.

---

## Guía de migración

El SDK exponía una sola constante `AFI_ADDRESS` para Base. Con el rollout
multi-chain, ahora viene con `AFI_ADDRESSES` (un `Record<chainId, Address>`);
la constante antigua quedó eliminada.

```typescript
// Antes
import { AFI_ADDRESS } from "@afi-run/sdk"
const router = AFI_ADDRESS

// Después
import { AFI_ADDRESSES } from "@afi-run/sdk"
const router = AFI_ADDRESSES[8453]              // Base
const arbRouter = AFI_ADDRESSES[42161]          // Arbitrum
```

El mismo formato aplica a `ROUTE_QUOTER_ADDRESSES` y `NMR_ADDRESSES`. Indexa
por chain ID — `client.chainId()` devuelve el valor que necesitas usar en
runtime.

Otras superficies que llegaron junto al multi-chain:

- **Encoders de admin** — `encodeAfiPause`, `encodeAfiSetFeeBps`, `encodeAfiAddRule`, … en `afi-admin.ts`. Úsalos desde un wallet connector cuando la clave de owner vive en un hardware/multisig.
- **Encoders de NMR** — `encodeNMRRequestOperation`, `encodeNMRSwap`, `encodeNMRLoan`, `encodeNMRSweepProfit`, `encodeNMRSetTreasury` en `nmr.ts`.
- **Parsers de evento** — 13 parsers tipados en `events.ts` (incl. `parseFlashLoanFailedWithData` y `parseNmrSwapExecuted`); mira [Indexación de eventos](#indexación-de-eventos).
- **Builders de step** — helpers `buildXxxStep(...)` por DEX; mira [Builders de step por DEX](#builders-de-step-por-dex).
- **Endpoints HTTP del quoter** — `findArbitrage`, `findPath`, `getRoutes`, `getLiquidationCandidates`, `liquidate`, `priceQuote`, `quoteDex`.

Ningún método existente cambió de firma — `client.swap()`, `client.quote()`,
`client.executeSwap()`, todos los helpers y todos los tipos de error siguen
idénticos.

---

## Directorio de ejemplos

| Archivo / carpeta                      | Qué muestra |
|----------------------------------------|-------------|
| `examples/nodejs/1-list-tokens.ts`     | Listar tokens activos en Base y BSC |
| `examples/nodejs/2-get-quote.ts`       | Builder con todas las opciones (priceBase, dexs, Token objects) |
| `examples/nodejs/3-execute-swap.ts`    | Cotizar → revisar → ejecutar (flujo recomendado para usuario) |
| `examples/nodejs/4-full-flow.ts`       | `.execute()` en una llamada |
| `examples/nodejs/5-approve-only.ts`    | Etapas: tokenInfo → hasAllowance → approve → simulate → submit → wait |
| `examples/nodejs/6-operator-batch.ts`  | `swapFor` + `batchSwapFor` para usuarios pre-aprobados |
| `examples/nodejs/7-nmr-arbitrage.ts`   | Ciclo de arbitraje flash-loan vía `executeNMRArbitrage` |
| `examples/nodejs/8-nmr-loan.ts`        | Arbitraje financiado por el usuario vía `nmrLoanArbitrage` |
| `examples/nodejs/9-admin-governance.ts`| Pause, fee bps, reglas — flujos exclusivos del owner |
| `examples/nodejs/10-event-indexer.ts`  | `getLogs` + los 11 parsers de evento |
| `examples/go/list-tokens/`             | Listar tokens activos |
| `examples/go/get-quote/`               | Functional options |
| `examples/go/execute-swap/`            | Cotizar → revisar → ejecutar |
| `examples/go/full-flow/`               | `Swap()` en una llamada |
| `examples/go/approve-only/`            | Flujo por etapas |
| `examples/go/operator-batch/`          | `SwapFor` + `BatchSwapFor` |
| `examples/go/nmr-arbitrage/`           | Ciclo de arbitraje flash-loan |
| `examples/go/nmr-loan/`                | Arbitraje financiado por el usuario |
| `examples/go/admin-governance/`        | Flujos exclusivos del owner |
| `examples/go/event-indexer/`           | Parsers de evento |

Correr ejemplos TS:

```bash
cd nodejs && npm install
npx ts-node ../examples/nodejs/1-list-tokens.ts
```

Correr ejemplos Go:

```bash
cd examples/go && go mod tidy
go run ./list-tokens
```

O compila todos los ejemplos en `bin/` (ignorado por git) vía el Makefile:

```bash
cd examples/go
make build              # compila cada ejemplo en bin/<nombre>
make run EX=get-quote   # o corre uno directo
make list               # muestra los ejemplos descubiertos
```

---

## Desarrollo

### Build desde el código fuente

```bash
# TypeScript
cd nodejs
npm install
npm run build       # genera en dist/
npm run typecheck   # tsc --noEmit
npm test            # vitest

# Go
cd go
go mod tidy
go build ./...
go test ./...
```

### Layout del proyecto

```
afi-sdk/
├── nodejs/          ── @afi-run/sdk (TypeScript)
│   ├── src/
│   │   ├── client.ts, builder.ts          ── client público + builder de cotización
│   │   ├── token.ts, multicall.ts         ── lecturas ERC-20 + Multicall3
│   │   ├── swap.ts, quoter.ts             ── pipelines de swap + cotización
│   │   ├── address.ts, slippage.ts        ── helpers de DX
│   │   ├── serialize.ts, explorer.ts      ── helpers JSON + URL
│   │   ├── errors.ts, types.ts            ── clases de error + tipos públicos
│   │   ├── constants.ts, utils.ts         ── ABIs, direcciones, unidades
│   │   └── index.ts                       ── exports públicos
│   └── src/__tests__/                     ── 159 tests unitarios
├── go/              ── github.com/afi-run/sdk/go
│   ├── client.go, options.go              ── client público + functional options
│   ├── token.go, multicall.go             ── lecturas ERC-20 + Multicall3
│   ├── swap.go, quoter.go                 ── pipelines de swap + cotización
│   ├── address.go, slippage.go            ── helpers de DX
│   ├── serialize.go, explorer.go          ── helpers JSON + URL
│   ├── errors.go, types.go                ── tipo de error + tipos públicos
│   └── *_test.go                          ── 159 tests unitarios
└── examples/        ── ejemplos end-to-end ejecutables
```

### Git hooks — `pre-push` replica el CI localmente

El repo viene con un hook de pre-push que corre los mismos checks del CI antes
de cada `git push`, así detectas problemas en segundos en lugar de esperar a
que un PR se ponga verde.

```bash
# Instalación única (define core.hooksPath = scripts/git-hooks)
bash scripts/install-hooks.sh
```

En `git push`, el hook detecta qué subproyectos cambiaron y corre:

| Subproyecto | Pasos |
|-------------|-------|
| **Node.js** (cuando `nodejs/` cambió) | `typecheck` · `vitest run --coverage` (≥95% stmts/lines/fns, ≥90% branches) · `npm audit --audit-level=high` |
| **Go** (cuando `go/` o `examples/go/` cambió) | `go vet` · `make test-coverage` (≥95%) · `go build` ejemplos · `govulncheck` (lo instala si falta) |

Atajos:

```bash
SKIP_PRE_PUSH=1 git push      # saltar esta vez
PRE_PUSH_ALL=1  git push      # ignorar detección de paths, correr todo
git config --unset core.hooksPath   # remover el hook por completo
```

### Estrategia de testing

El SDK está **totalmente unit-testeado** sin dependencias externas: las
llamadas RPC y HTTP están mockeadas. Correlas con `npm test` o `go test ./...`.

En tus propios tests, mockea la frontera `AfiClient` / `*afi.Client` — el SDK
ya confía en que el RPC devuelve respuestas válidas, y ese es el punto más
limpio para tus fixtures.

---

## Licencia

MIT © contribuidores de AFI Run. Mira [LICENSE](./LICENSE) para detalles.
