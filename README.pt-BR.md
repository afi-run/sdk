# AFI SDK

> SDK profissional para execução de swaps em redes EVM através do [Protocolo AFI](https://afi.run).

Construa interfaces de swap, bots de trading, ferramentas analíticas e indexers
sem reimplementar descoberta de rota, matemática de slippage, fluxo de
allowance, buffer de gas, decodificação de revert ou parsing de eventos.

| | |
|---|---|
| **Linguagens** | TypeScript (Node.js 18+) · Go 1.21+ |
| **Redes (cotação)** | Base · BSC · Arbitrum · Ethereum · Unichain |
| **Redes (execução)** | Todas as acima (chain ID detectado do RPC) |
| **Traduções** | [English](./README.md) · [Español](./README.es.md) |
| **Licença** | MIT |

---

## Por que esse SDK

- **Swap em uma chamada** — `client.swap()` encadeia cotação → checagem de saldo → approve → simulate → submit → wait.
- **Fluxo em etapas** — todo passo também é exposto individualmente para controle granular de UI.
- **Seguro por padrão** — allowance exato, `minOut` aplicado on-chain, simulação antes do broadcast.
- **Cotações multi-chain** — cote em 5 redes EVM usando um único client.
- **Ergonomia operacional** — health checks, logs estruturados, serialização JSON, leituras via multicall, gas buffer configurável, confirmações, timeouts.

---

## Sumário

- [Requisitos](#requisitos)
- [Instalação](#instalação)
- [Início rápido](#início-rápido)
- [Conceitos centrais](#conceitos-centrais)
- [Referência da API](#referência-da-api)
  - [Construção do client](#construção-do-client)
  - [Operações de leitura](#operações-de-leitura)
  - [Builder de cotação](#builder-de-cotação)
  - [Operações de escrita (exigem signer)](#operações-de-escrita-exigem-signer)
  - [Utilities de transação](#utilities-de-transação)
  - [Configuração](#configuração)
- [Helpers](#helpers)
- [Logs e diagnóstico](#logs-e-diagnóstico)
- [Tratamento de erros](#tratamento-de-erros)
- [Modelo de segurança](#modelo-de-segurança)
- [Receitas](#receitas)
- [Redes e constantes](#redes-e-constantes)
- [Diretório de exemplos](#diretório-de-exemplos)
- [Desenvolvimento](#desenvolvimento)
- [Licença](#licença)

---

## Requisitos

| Runtime    | Mínimo   | Recomendado |
|------------|----------|-------------|
| Node.js    | 18.x     | 20.x LTS    |
| TypeScript | 5.0      | última      |
| Go         | 1.21     | 1.22+       |

Você também precisa de um endpoint RPC HTTP para cada rede em que vai ler ou
executar. Provedores públicos (Ankr, Alchemy, Infura, drpc, …) funcionam em
desenvolvimento; **use um plano pago (ou seu próprio nó) em produção** para
evitar timeouts no quoter e reverts por rate-limit.

---

## Instalação

### TypeScript / Node.js

```bash
npm install @afi-run/sdk     # ou: pnpm add @afi-run/sdk · yarn add @afi-run/sdk
```

Até o pacote ser publicado no npm:

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

## Início rápido

### TypeScript — cotação somente leitura

```typescript
import { AfiClient, NETWORK, formatUnits } from "@afi-run/sdk"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/SUA_CHAVE",
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
console.log(`Criada em:   ${new Date(quote.createdAt).toISOString()}`)
```

### TypeScript — swap em uma chamada

```typescript
client.connect("0xSUA_CHAVE_PRIVADA")

const result = await client
  .quote(USDC, WETH, "500")
  .slippage(0.5)
  .execute({ confirmations: 1 })

console.log(`Tx:         ${client.txUrl(result.txHash)}`)
console.log(`Recebido:   ${formatUnits(result.amountOut, 18)} WETH`)
console.log(`Gas usado:  ${result.gasUsed}`)
```

### Go — cotação somente leitura

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
        RPCURL: "https://rpc.ankr.com/base/SUA_CHAVE",
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

### Go — swap em uma chamada

```go
client.Connect("SUA_CHAVE_PRIVADA")

result, err := client.Swap(ctx,
    afi.From(usdc, afi.WETH, "500"),
    afi.WithSlippage(0.5),
)
if err != nil {
    log.Fatal(err)
}
url, _ := client.TxURL(result.TxHash.Hex())
fmt.Printf("Tx:        %s\n", url)
fmt.Printf("Recebido:  %s WETH\n", afi.FormatUnits(result.AmountOut, 18))
```

---

## Conceitos centrais

### Ciclo de vida do swap

Todo swap passa por cinco etapas. `executeSwap(quote)` roda as etapas 2–5
atomicamente; cada uma é também exposta para fluxos em etapas (UI passo a passo).

```
1. Cotação        ─ POST /quoter — calcula rota, slippage, minOut
2. Saldo          ─ ERC20.balanceOf(owner) ≥ amountIn
3. Aprovação      ─ ERC20.approve(AFI, amountInWei)        (pulada se o allowance basta)
4. Simulação      ─ eth_call AFI.swap(...)                 (falha rápido se revertaria)
5. Envio + espera ─ broadcast e espera confirmação(ões)
```

### Modo leitura vs modo signer

O client tem dois modos, escolhidos por ter ou não chave privada:

- **Leitura** — `quote`, `tokenInfo`, `getBalance`, `getEthBalance`,
  `getAllowance`, `hasAllowance`, `getFeeBps`, `chainId`, `detectNetwork`,
  `health`, `txUrl`, `addressUrl`.
- **Signer** (adiciona) — `approve`, `simulate`, `submitSwap`, `executeSwap`,
  `swap`, `estimateSwapCost`.

Métodos de leitura continuam disponíveis após `connect()`. Métodos de escrita
lançam `NoSignerError` quando a chave privada não está configurada.

### Modelo de gas buffer

Todas as transações de escrita (approve + swap) multiplicam o resultado de
`eth_estimateGas` por `(1 + gasBufferPercent / 100)`. O padrão é **+15 %**.
Configure com `gasBufferPercent` na construção do client, ou sobrescreva em
runtime com `setGasBufferPercent(n)`. Passe `0` para desativar.

O buffer só afeta o gas enviado a `writeContract` / `SendTransaction`, nunca o
preço (o `maxFeePerGas` continua sendo `baseFee * 2 + tip`).

### Slippage e garantia de `minOut`

Toda `Quote` carrega `minOutWei` — o mínimo que o router AFI aceita on-chain.
O contrato reverte se a execução fosse entregar menos, então o usuário nunca
recebe menos que esse valor. O SDK recusa cotações com `minOutWei = 0`.

Slippage é em porcentagem (`0.5` = 0,5 %) e aplicado pelo quoter. Use o helper
[`calculateMinOut`](#calculadora-de-slippage) se precisar derivar do lado cliente.

### Builder vs functional options

- **TypeScript** — `client.quote(...)` retorna um `QuoteBuilder` fluente.
  Encadeie `.slippage()`, `.maxHops()`, `.network()`, etc., e termine com `.get()` ou `.execute()`.
- **Go** — `client.GetQuote(ctx, opts...)` aceita functional options
  (`afi.From`, `afi.WithSlippage`, `afi.OnNetwork`, …).

Ambos cobrem a mesma configuração; use o que combina com o estilo do seu codebase.

---

## Referência da API

### Construção do client

#### TypeScript

```typescript
new AfiClient(config: AfiConfig)

interface AfiConfig {
  rpcUrl:             string             // obrigatório — RPC da rede de execução
  privateKey?:        Hex                // opcional — habilita modo signer
  gasBufferPercent?:  number             // padrão: 15 — % acima do estimateGas
  logger?:            Logger             // opcional — callback de diagnóstico
}
```

#### Go

```go
afi.NewClient(cfg afi.Config) (*afi.Client, error)

type Config struct {
    RPCURL           string  // obrigatório
    PrivateKey       string  // opcional — hex com ou sem 0x
    GasBufferPercent uint    // padrão: 15 — zero usa o padrão; SetGasBufferPercent(0) desativa
    Logger           Logger  // opcional
}
```

`Close()` (Go) fecha a conexão RPC subjacente.

---

### Operações de leitura

| Método | Retorno | Descrição |
|---|---|---|
| `getTokens(network?)` / `GetTokens(ctx, network?)` | `Token[]` | Tokens ativos. Cache por rede. |
| `findToken(symbol, network?)` / `FindToken(ctx, symbol, network?)` | `Token \| null` | Lookup case-insensitive. Usa o cache. |
| `clearTokensCache(network?)` / `ClearTokensCache(network?)` | `void` | Invalida o cache (todas as redes ou só uma). |
| `getFeeBps()` / `GetFeeBps(ctx)` | `number` / `uint16` | Taxa atual do protocolo no contrato AFI. |
| `tokenInfo(token, owner?)` / `TokenInfo(ctx, token, owner)` | `TokenInfo` | symbol/name/decimals (+ balance/allowance) em **um multicall**. |
| `tokenInfoBatch(tokens, owner?)` / `TokenInfoBatch(ctx, tokens, owner)` | `TokenInfo[]` | O mesmo para N tokens em um único multicall. |
| `getBalance(token, owner?)` / `GetBalance(ctx, token, owner?)` | `bigint` / `*big.Int` | Saldo ERC-20. |
| `getEthBalance(owner?)` / `GetETHBalance(ctx, owner?)` | `bigint` / `*big.Int` | Saldo de ETH nativo. |
| `getAllowance(token, owner?)` / `GetAllowance(ctx, token, owner?)` | `bigint` / `*big.Int` | Quanto o router AFI pode gastar pelo `owner`. |
| `hasAllowance(token, amount, owner?)` / `HasAllowance(ctx, token, amount, owner?)` | `boolean` | Conveniência: `getAllowance >= amount`. |
| `chainId()` / `ChainID(ctx)` | `number` / `*big.Int` | Chain ID lido do RPC (cacheado). |
| `detectNetwork()` / `DetectNetwork(ctx)` | `Network \| null` | Mapeia o chain ID para uma `Network` conhecida. |
| `health()` / `Health(ctx)` | `HealthCheck` | Probe paralelo de RPC + API. |
| `estimateSwapCost(quote)` / `EstimateSwapCost(ctx, quote)` | `SwapCostEstimate` | Projeta o custo sem enviar tx. **Exige signer.** |

`owner` omitido usa a carteira conectada. Em Go, passe `common.Address{}` para
o mesmo efeito. `TokenInfo` em TS aceita `"self"` como atalho.

#### Token

```typescript
interface Token {
  address:  Address     // 0x… 20 bytes
  symbol:   string      // ex: "USDC"
  decimals: number      // ex: 6
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
  owner?:     Address    // só quando um owner foi passado
  balance?:   bigint     // saldo ERC-20 do owner
  allowance?: bigint     // allowance concedida ao AFI pelo owner
}
```

#### HealthCheck

```typescript
interface HealthEndpoint {
  ok:          boolean
  durationMs:  number
  detail?:     string    // "chainId=8453" no RPC, "ok" ou "HTTP 503" na API
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
  gas:           bigint   // eth_estimateGas cru
  gasWithBuffer: bigint   // gas * (1 + gasBufferPercent/100)
  gasPriceWei:   bigint   // maxFeePerGas que o SDK usaria = baseFee * 2 + tip
  totalWei:      bigint   // gasWithBuffer * gasPriceWei
  totalEth:      string   // totalWei formatado em ETH (18 decimais)
}
```

---

### Builder de cotação

#### TypeScript

```typescript
client.quote(tokenIn: Address | Token, tokenOut: Address | Token, amountIn: string): QuoteBuilder
```

| Método            | Padrão      | Descrição |
|-------------------|-------------|-----------|
| `.slippage(v)`    | `0.5`       | Tolerância de slippage em % |
| `.maxHops(n)`     | `2`         | Máximo de hops |
| `.network(n)`     | `BASE`      | Rede alvo |
| `.priceBase(s)`   | —           | Preenche `tokenInBasePrice` / `tokenOutBasePrice` |
| `.dexs(...)`      | —           | Restringe DEXes |
| `.blockNumber(n)` | `"latest"`  | Cotar contra um bloco específico |
| `.rpcUrls(...)`   | RPC client  | Sobrescreve endpoints usados pelo quoter |
| `.get()`          | —           | Busca e retorna `Quote` |
| `.execute(opts?)` | —           | Busca + executa. Exige signer. |

#### Go

```go
client.GetQuote(ctx context.Context, opts ...QuoteOption) (*Quote, error)
client.Swap(ctx context.Context, opts ...QuoteOption) (*SwapResult, error)
```

| Option                   | Padrão       | Descrição |
|--------------------------|--------------|-----------|
| `From(in, out, amount)`  | **obrigatório** | Par + valor de entrada |
| `WithSlippage(v)`        | `0.5`        | Slippage em % |
| `WithMaxHops(n)`         | `2`          | Máximo de hops |
| `OnNetwork(n)`           | `NetworkBase`| Rede alvo |
| `WithPriceBase(s)`       | —            | Igual `.priceBase` |
| `WithDexs(...)`          | —            | Restringe DEXes |
| `WithBlockNumber(n)`     | `"latest"`   | Bloco específico |
| `WithRpcUrls(...)`       | RPC client   | Sobrescreve endpoints do quoter |

#### Quote

```typescript
interface Quote {
  tokenIn:           Address    // token de entrada
  tokenOut:          Address    // token de saída
  amountIn:          string     // valor de entrada legível
  amountOut:         string     // estimativa de saída legível
  minOut:            string     // saída mínima após slippage (legível)
  amountInWei:       bigint     // entrada exata — passe a approve()
  amountOutWei:      bigint     // saída estimada (Wei)
  minOutWei:         bigint     // mínimo aplicado on-chain (Wei)
  steps:             Hex        // rota codificada — passada a AFI.swap()
  path:              Address[]  // endereços de tokens no caminho
  hops:              Hop[]      // detalhamento por hop
  slippage:          number     // slippage aplicado em %
  feeBps:            number     // taxa do protocolo na cotação
  tokenInPrice:      string     // preço de tokenIn em unidades de tokenOut
  tokenOutPrice:     string     // preço de tokenOut em unidades de tokenIn
  tokenInBasePrice?: string     // preenchido por priceBase()
  tokenOutBasePrice?: string    // preenchido por priceBase()
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
  type:          string    // protocolo da pool, ex: "v3", "v2"
  kind:          string    // motor de roteamento
  routeId:       number
  weight:        number
}
```

---

### Operações de escrita (exigem signer)

#### `connect(privateKey)` / `Connect(privateKey)`

Anexa um signer. Aceita hex com ou sem o prefixo `0x`.

```typescript
client.connect("0x…")
const c = new AfiClient({ rpcUrl, privateKey: "0x…" })
```

```go
err := client.Connect("…")
```

`client.address()` (TS) / `client.Address()` (Go) retorna o endereço derivado da
chave, ou o endereço zero quando não há signer.

#### `approve(token, amountWei)` / `Approve(ctx, token, amountWei)`

Envia um approve do valor exato para o router AFI. Retorna um `PendingTx` (hash
disponível imediatamente) ou **null** quando o allowance existente já é
suficiente — economizando uma transação.

Para tokens estilo USDT, o SDK reseta o allowance para zero primeiro. Se o reset
em si falhar (e o approve subsequente também), os dois erros aparecem na
`ApprovalError` resultante.

```typescript
const pending = await client.approve(quote.tokenIn, quote.amountInWei)
if (pending) {
  console.log("Tx de aprovação:", pending.txHash)
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

Faz `eth_call` no router AFI. Resolve (ou retorna `nil`) em sucesso. Lança
`SimulationFailedError` (TS) ou retorna `*AfiError{Code:"SIMULATION_FAILED"}`
(Go) com o motivo do revert quando o swap iria reverter. **Nenhuma tx é
enviada em qualquer cenário.**

```typescript
try {
  await client.simulate(quote)
} catch (e) {
  if (isSimulationFailedError(e)) console.error("reverteria:", e.reason)
}
```

```go
if err := client.Simulate(ctx, quote); err != nil {
    log.Println("reverteria:", err)
}
```

#### `submitSwap(quote)` / `SubmitSwap(ctx, quote)`

Envia a tx de swap sem esperar confirmação. Retorna um `PendingSwap` cujo
`wait(opts?)` bloqueia até confirmar.

#### `executeSwap(quote, opts?)` / `ExecuteSwap(ctx, quote, opts?)`

Roda a sequência completa — saldo → approve → simulate → submit → wait.
Retorna quando o swap está confirmado.

```typescript
interface ExecuteOptions {
  confirmations?: number    // padrão: 1
  timeoutMs?:     number    // padrão: sem timeout
}
```

```go
type ExecuteOptions struct {
    Confirmations uint64
    TimeoutMs     int64
}
```

#### `swap(opts)` / `Swap(ctx, opts...)`

Conveniência: cota e executa. Use o fluxo em etapas ou `executeSwap(quote, opts)`
quando precisar de confirmations/timeout ou confirmação manual entre cotação
e execução.

#### `estimateSwapCost(quote)` / `EstimateSwapCost(ctx, quote)`

Projeta o custo de gas sem enviar tx. Retorna [`SwapCostEstimate`](#swapcostestimate).
Útil para mostrar "taxa de rede estimada" antes do usuário assinar.

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
  amountIn:    bigint     // amountIn real do evento SwapExecuted
  amountOut:   bigint     // amountOut real do evento SwapExecuted
  tokenIn:     Address
  tokenOut:    Address
  gasUsed:     bigint
}

interface TxReceipt {
  blockNumber: bigint
  gasUsed:     bigint
}

interface WaitForTxOptions {
  confirmations?: number   // padrão: 1
  timeoutMs?:     number   // padrão: sem timeout
}
```

---

### Utilities de transação

#### `waitForTx(hash, opts?)` / `WaitForTx(ctx, hash, opts?)`

Faz polling até a tx atingir as confirmações desejadas. Útil para hashes
obtidos fora do SDK (persistidos em DB, vindos de outro serviço, em fila).

```typescript
const receipt = await client.waitForTx("0x…", { confirmations: 2, timeoutMs: 30_000 })
```

```go
receipt, err := client.WaitForTx(ctx, "0x…", afi.WaitForTxOptions{
    Confirmations: 2, TimeoutMs: 30_000, PollIntervalMs: 1_000,
})
```

#### `parseSwapResult(receipt)` / `ParseSwapResult(receipt)`

Decodifica o evento `SwapExecuted` de qualquer receipt. Retorna `null` / `nil`
quando não há log `SwapExecuted` (a tx não era um swap AFI).

```typescript
import { parseSwapResult } from "@afi-run/sdk"

const result = parseSwapResult(receipt) // SwapResult | null
```

```go
result, err := afi.ParseSwapResult(receipt) // nil quando não tem SwapExecuted no receipt
```

Use em indexers, ferramentas de replay, jobs em fila que guardam o hash para
reconciliar depois, e testes ponta-a-ponta.

---

### Configuração

| Método | Descrição |
|---|---|
| `setApiUrl(url)` / `SetApiURL(url)` | Sobrescreve a URL base da API AFI (padrão `https://rpc.afi.run`). |
| `setGasBufferPercent(n)` / `SetGasBufferPercent(n)` | Sobrescreve o buffer em runtime. `0` desativa. |
| `setLogger(fn)` / `SetLogger(fn)` | Anexa ou substitui o logger de diagnóstico. |
| `clearTokensCache(network?)` / `ClearTokensCache(network?)` | Força refetch na próxima `getTokens()`. |

---

## Helpers

### Utilities de endereço

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
afi.IsAddress(s)          // exige prefixo "0x" (matching viem/ethers)
afi.Checksum(s)           // string EIP-55
afi.IsZeroAddress(s)
afi.EqualAddresses(a, b)
afi.ZeroAddress           // common.Address{}
afi.ZeroAddressHex        // "0x00…00"
```

### Calculadora de slippage

```typescript
import { calculateMinOut, applySlippage } from "@afi-run/sdk"

const minOut = calculateMinOut(quote.amountOutWei, 0.5)  // 0.5% off, arredonda para baixo
```

```go
minOut := afi.CalculateMinOut(quote.AmountOutWei, 0.5)
```

`slippagePct` é em porcentagem (`0.5` = 0,5 %). Negativos viram 0. Valores
≥ 100 retornam 0.

### Conversão de unidades

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
txUrl(hash, NETWORK.BASE, "https://meu-explorer")   // base customizado
```

```go
url, _ := client.TxURL(result.TxHash.Hex())
addr, _ := afi.AddressURL(walletAddr, afi.NetworkArbitrum)
```

Padrões em `NETWORK_EXPLORERS` / `afi.NetworkExplorers`, sobrescrevíveis em
runtime.

### Staleness de cotação

```typescript
import { isQuoteStale } from "@afi-run/sdk"

if (isQuoteStale(quote, 30)) {   // mais velho que 30 segundos
  quote = await client.quote(...).get()
}
```

```go
if quote.IsStale(30) {
    quote, _ = client.GetQuote(ctx, ...)
}
```

Cotações envelhecem rápido (poucos segundos em pares voláteis). Sempre reconote
antes do broadcast em fluxos lentos (carteira hardware, multi-sig, revisão manual).

### Decodificação de custom errors (`decodeRevertReason`, campos `decoded` nos erros)

O SDK vem com os **9 custom errors do router AFI** pré-registrados (verificados
no Basescan), mais os do OpenZeppelin (`Ownable*`, `ReentrancyGuardReentrantCall`)
e os built-in do Solidity (`Error(string)`, `Panic(uint256)`). Reverts são
decodificados automaticamente; o resultado estruturado fica anexado ao erro lançado.

```typescript
try {
  await client.simulate(quote)
} catch (e) {
  if (isSimulationFailedError(e) && e.decoded) {
    // e.decoded = { name: "InsufficientFunds", signature: "InsufficientFunds(uint256)", args: [100n] }
    if (e.decoded.name === "InsufficientFunds") {
      toast.error(`Pool só tem ${e.decoded.args[0]} disponível`)
    }
  }
}
```

```go
err := client.Simulate(ctx, quote)
var afiErr *afi.AfiError
if errors.As(err, &afiErr) && afiErr.Decoded != nil {
    if afiErr.Decoded.Name == "InsufficientFunds" {
        log.Printf("pool só tem %s disponível", afiErr.Decoded.Args[0])
    }
}
```

#### Erros decodificados que vêm no SDK

| Erro                                  | Origem        |
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

#### Registrar os erros do seu próprio contrato

```typescript
import { registerCustomErrors, decodeRevertReason } from "@afi-run/sdk"

registerCustomErrors([
  { type: "error", name: "MyContractError", inputs: [
    { name: "code", type: "uint256" },
    { name: "msg",  type: "string" },
  ]},
])

// Decodificar revert data cru manualmente
const decoded = decodeRevertReason("0x…")
// ou simplesmente confiar que ele virá anexado aos próximos erros lançados
```

```go
// Parse qualquer ABI com definições de error e registre globalmente.
a, _ := abi.JSON(strings.NewReader(`[{"type":"error","name":"MyContractError", ...}]`))
afi.RegisterCustomErrors(a)

decoded := afi.DecodeRevertReason(rawHexBytes) // *afi.DecodedRevert
```

---

### Taxa da transação nos resultados

`SwapResult` e `TxReceipt` agora expõem o custo real pago:

```typescript
const result = await client.executeSwap(quote)
console.log(`Custo: ${result.feeEth} ETH (${result.feeWei} wei @ ${result.effectiveGasPrice} wei/gas)`)
```

Mesmos campos no Go:

```go
fmt.Printf("Custo: %s ETH\n", result.FeeETH)
```

Receipts retornados por `waitForTx`, `pending.wait()`, e `wait()` de
`approve`/`revoke` carregam a taxa.

---

### `getTxStatus(hash)` — status sem bloquear

Retorna imediatamente o estado atual da tx — útil para indicadores de UI em
polling onde bloquear num receipt seria desastre.

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

### `getTokenPrice(in, out, opts?)` — consulta de preço rápida

Lookup leve de taxa de câmbio entre um par sem comprometer-se com o swap.

```typescript
const { price, inverse } = await client.getTokenPrice(USDC, WETH)
// price   = "0.00031" (1 USDC em WETH)
// inverse = "3225"    (1 WETH em USDC)

// Sobrescrevendo padrões:
await client.getTokenPrice(USDC, WETH, { amount: "1000", slippage: 1.0, network: NETWORK.BSC })
```

```go
p, _ := client.GetTokenPrice(ctx, usdc, weth)
// p.Price, p.Inverse
```

---

### Gerenciamento de nonce — `getNonce`, `useManagedNonce`

Para bots que enviam múltiplos swaps em paralelo sem esperar entre eles.

```typescript
// Leitura única
const n = await client.getNonce()

// Modo gerenciado (recomendado para bots)
await client.useManagedNonce()    // sincroniza com a chain e mantém um contador local
await Promise.all([
  client.executeSwap(quote1),
  client.executeSwap(quote2),
  client.executeSwap(quote3),     // cada um pega um nonce único, sem race
])

// Em caso de erro / fork / replacement
await client.resetManagedNonce()  // re-sincroniza com a chain

// Override por chamada
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

Quando o contador local desincronizar (tx rejeitada, replacement), chame
`resetManagedNonce()` / `ResetManagedNonce(ctx)` para re-sincronizar.

---

### `preflight(quote)` — checagem combinada de prontidão

Roda balance + allowance + simulate **sem enviar tx** e retorna um relatório
estruturado para a UI mostrar "pronto para fazer swap".

```typescript
const report = await client.preflight(quote)
if (!report.canExecute) {
  for (const p of report.problems) console.error(`${p.code}: ${p.message}`)
} else if (report.needsApproval) {
  showButton("Aprovar & Trocar")
} else {
  showButton("Trocar")
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

`canExecute = problems.length === 0` — `needsApproval` é informativo porque
`executeSwap` cuida do approve automaticamente.

---

### Transações pré-codificadas (`encodeSwap`, `encodeApprove`, `encodeRevoke`)

Para frontends onde a chave privada vive numa carteira do usuário (Wagmi,
RainbowKit, MetaMask, Frame, hardware wallet, Safe SDK), construa o calldata
com o SDK e submeta pelo connector.

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

Os três também estão expostos como métodos do client (`client.encodeSwap(quote)`,
etc.) quando já há um client configurado.

---

### Revogar allowance — `revoke(token)` / `Revoke(ctx, token)`

Envia `approve(AFI, 0)` zerando a permissão do router. Retorna `null` / `nil`
quando o allowance já está em zero. Use como cleanup de segurança pós-swap.

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

Junta N reads arbitrárias em uma chamada RPC via Multicall3. Útil para qualquer
batch além de `tokenInfo` — preço de pools, seus próprios contratos, estado de
DEX custom.

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
// O ABI do Multicall3 está em afi.Multicall3ABIJSON para uso de baixo nível.
calls := []afi.Multicall3Call{ /* … */ }
results, err := client.Multicall(ctx, calls)
```

---

### Atualizar cotação velha — `refreshQuote(quote)` / `RefreshQuote(ctx, quote)`

Refaz a cotação usando os parâmetros originais (network, slippage, maxHops,
priceBase, dexs). Conveniência para fluxos lentos (confirmação em hardware
wallet, revisão multi-sig) onde o builder original já se foi.

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

`tokenInfo` / `tokenInfoBatch` guardam `(symbol, name, decimals)` num cache em
memória — esses valores nunca mudam para um token ERC-20, então o segundo
lookup de metadata custa **zero RPC**. Quando há `owner`, só balance/allowance
são re-buscados nas chamadas subsequentes.

Limpe o cache com `clearTokenMetadataCache()` (TS) / `ClearTokenMetadataCache()`
(Go) se trocar de provedor RPC e quiser revalidar os tokens.

---

### Serialização JSON

`Quote`, `SwapResult` e `TokenInfo` têm campos `bigint` / `*big.Int` que
quebram `JSON.stringify` e perdem precisão em `json.Marshal`. O SDK oferece
helpers de round-trip (bigints viram strings base-10).

```typescript
import {
  bigintReplacer,
  quoteToJSON,    quoteFromJSON,
  swapResultToJSON, swapResultFromJSON,
  tokenInfoToJSON,  tokenInfoFromJSON,
} from "@afi-run/sdk"

// Salvar
await db.put(`quote:${id}`, JSON.stringify(quoteToJSON(quote)))
// Alternativa genérica para objetos arbitrários:
JSON.stringify(qualquerObjeto, bigintReplacer)

// Restaurar
const restored = quoteFromJSON(await db.get(`quote:${id}`))
```

```go
// Quote / SwapResult / TokenInfo implementam MarshalJSON e UnmarshalJSON.
data, _ := json.Marshal(quote)        // bigints viram strings
var q afi.Quote
_ = json.Unmarshal(data, &q)
```

### ABIs exportadas

```typescript
import { AFI_ABI, ERC20_ABI, MULTICALL3_ABI } from "@afi-run/sdk"
// drop-in para viem.readContract / writeContract / parseEventLogs
```

```go
// Strings JSON cruas — passe para abi.JSON(strings.NewReader(...)).
afi.AFIABIJSON
afi.ERC20ABIJSON
afi.Multicall3ABIJSON
```

---

## Logs e diagnóstico

Anexe um logger para capturar timing e desfecho das operações principais.

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

Troque em runtime com `setLogger(fn)` / `SetLogger(fn)`. Passe `undefined` /
`nil` para desativar.

`health()` / `Health(ctx)` sonda RPC (chain ID) e API AFI em paralelo:

```typescript
const h = await client.health()
if (!h.rpc.ok || !h.api.ok) {
  console.error("não pronto:", h)
  process.exit(1)
}
```

---

## Tratamento de erros

Todos os erros lançados derivam de `AfiError` (TS) / `*AfiError` (Go). O `Code`
identifica a classe; a `Message` é amigável; alguns códigos anexam campos
extras.

### Referência de códigos de erro

| Código | Quando | Campos extras |
|---|---|---|
| `NO_SIGNER` | Método de escrita chamado sem `connect()`. | — |
| `INSUFFICIENT_BALANCE` | Saldo de tokenIn abaixo do necessário. | `token`, `owner`, `symbol`, `decimals`, `balance`, `required` |
| `APPROVAL_FAILED` | `approve()` (ou o reset estilo USDT) reverteu. | — |
| `SIMULATION_FAILED` | `eth_call` de `AFI.swap(...)` reverteu antes de qualquer tx ser enviada. | `reason`, `revertData?` (TS) |
| `QUOTE_FAILED` | A API do quoter retornou erro (sem rota, validação, …). | — |
| `SWAP_REVERTED` | A tx de swap reverteu on-chain, ou `estimateGas` falhou. | `reason` (TS) |

Quando `INSUFFICIENT_BALANCE` ocorre, o SDK faz **um multicall extra** para
anexar `symbol` e `decimals` — a mensagem fica
*"Insufficient USDC for 0xABcd…: have 0.5, need 1"* em vez de endereços crus.

### TypeScript — type guards

Prefira os guards a `instanceof` — eles sobrevivem a class shims e transpilação
em `esbuild`, `swc`, interop ESM↔CJS, etc.

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
    toast.error(`Reverteria: ${e.reason}`)
  } else if (isQuoteError(e)) {
    toast.error("Sem rota disponível.")
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
            log.Println("conecte um signer primeiro")
        case "INSUFFICIENT_BALANCE":
            log.Printf("faltam %s %s", new(big.Int).Sub(afiErr.Required, afiErr.Balance), afiErr.Symbol)
        case "SIMULATION_FAILED":
            log.Println("reverteria:", afiErr.Message)
        case "QUOTE_FAILED":
            log.Println("sem rota")
        case "APPROVAL_FAILED":
            log.Println("approve reverteu")
        case "SWAP_REVERTED":
            log.Println("revert on-chain")
        }
        return
    }
    log.Fatal(err) // erro de rede / encoding / programação
}
```

---

## Modelo de segurança

| Risco                         | Mitigação |
|-------------------------------|-----------|
| Bypass de slippage            | `minOutWei` vem sempre do quoter; valores zero são rejeitados client-side. |
| Aprovação excessiva           | Aprova sempre exatamente `amountInWei` — nunca `MAX_UINT256`. |
| Falha estilo USDT             | O SDK reseta para 0 antes de re-aprovar; falhas do reset são preservadas e exibidas se o approve subsequente também falhar. |
| Tx que reverteria             | `simulate` roda antes de todo `executeSwap` — falhas lançam sem gastar gas. |
| Race em allowance             | Allowance é relido on-chain após cada approve para confirmar. |
| Subestimativa de gas          | `eth_estimateGas` × `(1 + gasBufferPercent/100)` (padrão +15 %). |
| Mismatch de rede              | O signer lê o chain ID do RPC (Go); o SDK expõe `chainId()` para verificação (TS). |
| Cotações velhas               | `Quote.createdAt` é setado na captura. Use `isQuoteStale(quote, maxAge)` antes do envio. |
| ETH nativo passado por engano | O router não aceita ETH nativo; passe WETH. |

O endereço do contrato router e a taxa do protocolo são lidos on-chain em
toda cotação — o SDK não confia em valores cacheados aqui.

---

## Receitas

### Salvar uma cotação e restaurar depois

```typescript
import { quoteToJSON, quoteFromJSON, isQuoteStale } from "@afi-run/sdk"

await redis.set(`quote:${userId}`, JSON.stringify(quoteToJSON(quote)))
// …
const raw = await redis.get(`quote:${userId}`)
const quote = quoteFromJSON(raw!)
if (isQuoteStale(quote, 60)) {
  // refazer
}
```

### Pular approve quando o allowance já basta

```typescript
if (await client.hasAllowance(quote.tokenIn, quote.amountInWei)) {
  // sem approve
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

### Mostrar taxa estimada antes de assinar

```typescript
const cost = await client.estimateSwapCost(quote)
toast.info(`Taxa de rede estimada: ~${cost.totalEth} ETH`)
```

### Dashboard de portfólio — info batch para N tokens

```typescript
const tokens = await client.getTokens()
const infos = await client.tokenInfoBatch(
  tokens.filter(t => t.active).map(t => t.address),
  "self",
)
infos.forEach(i => console.log(`${i.symbol}: ${formatUnits(i.balance ?? 0n, i.decimals)}`))
```

### Fail-fast no startup

```typescript
const h = await client.health()
if (!h.rpc.ok || !h.api.ok) {
  console.error("SDK AFI não pronto", h)
  process.exit(1)
}
const net = await client.detectNetwork()
if (net !== "base") {
  console.error(`Esperava RPC para base, peguei ${net}`)
  process.exit(1)
}
```

### Bot esperando 2 confirmações

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

### Reprocessar ou indexar uma tx conhecida

```typescript
import { parseSwapResult } from "@afi-run/sdk"
const receipt = await publicClient.getTransactionReceipt({ hash })
const result = parseSwapResult(receipt)
if (result) await indexSwap(result)
```

---

## Redes e constantes

### Redes suportadas

| Rede        | Chain ID | Explorer padrão              | Constante TS           | Constante Go           |
|-------------|----------|------------------------------|------------------------|------------------------|
| Base        | 8453     | https://basescan.org         | `NETWORK.BASE`         | `afi.NetworkBase`      |
| BSC         | 56       | https://bscscan.com          | `NETWORK.BSC`          | `afi.NetworkBSC`       |
| Arbitrum    | 42161    | https://arbiscan.io          | `NETWORK.ARBITRUM`     | `afi.NetworkArbitrum`  |
| Ethereum    | 1        | https://etherscan.io         | `NETWORK.ETHEREUM`     | `afi.NetworkEthereum`  |
| Unichain    | 130      | https://uniscan.xyz          | `NETWORK.UNICHAIN`     | `afi.NetworkUnichain`  |

Sobrescreva explorers em runtime via `NETWORK_EXPLORERS` (TS) ou
`afi.NetworkExplorers` (Go).

### DEXes suportadas

```typescript
import { DEX } from "@afi-run/sdk"
// DEX.UNI_V3 · DEX.UNI_V4 · DEX.CAKE_V3 · DEX.AERODROME
// DEX.BALANCER · DEX.CURVE128 · DEX.CURVE256 · DEX.FLUID
```

```go
// afi.DexUniV3 · afi.DexUniV4 · afi.DexCakeV3 · afi.DexAerodrome
// afi.DexBalancer · afi.DexCurve128 · afi.DexCurve256 · afi.DexFluid
```

### Constantes de contrato

| Nome                  | Valor |
|-----------------------|-------|
| Router AFI (Base)     | `0xB8cC65321d169D55b93b4402D795701c6B308ce4` |
| WETH (Base)           | `0x4200000000000000000000000000000000000006` |
| Multicall3 (todas)    | `0xcA11bde05977b3631167028862bE2a173976CA11` |
| URL base da API       | `https://rpc.afi.run` |
| Gas buffer padrão     | `15` (%) |

```typescript
import {
  AFI_ADDRESS,
  WETH,
  MULTICALL3_ADDRESS,
  DEFAULT_GAS_BUFFER_PERCENT,
  NETWORK_CHAIN_IDS,
} from "@afi-run/sdk"
```

```go
afi.AfiAddress         // common.Address
afi.WETH               // common.Address
afi.Multicall3Address  // common.Address
afi.DefaultGasBufferPercent
afi.NetworkChainIDs    // map[Network]int64
```

---

## Diretório de exemplos

| Arquivo / pasta                        | O que mostra |
|----------------------------------------|--------------|
| `examples/nodejs/1-list-tokens.ts`     | Listar tokens ativos em Base e BSC |
| `examples/nodejs/2-get-quote.ts`       | Builder com todas as opções (priceBase, dexs, Token objects) |
| `examples/nodejs/3-execute-swap.ts`    | Cotar → revisar → executar (fluxo recomendado para usuário) |
| `examples/nodejs/4-full-flow.ts`       | `.execute()` em uma chamada |
| `examples/nodejs/5-approve-only.ts`    | Etapas: tokenInfo → hasAllowance → approve → simulate → submit → wait |
| `examples/go/list-tokens/`             | Listar tokens ativos |
| `examples/go/get-quote/`               | Functional options |
| `examples/go/execute-swap/`            | Cotar → revisar → executar |
| `examples/go/full-flow/`               | `Swap()` em uma chamada |
| `examples/go/approve-only/`            | Fluxo em etapas |

Executar exemplos TS:

```bash
cd nodejs && npm install
npx ts-node ../examples/nodejs/1-list-tokens.ts
```

Executar exemplos Go:

```bash
cd examples/go && go mod tidy
go run ./list-tokens
```

---

## Desenvolvimento

### Build a partir do código-fonte

```bash
# TypeScript
cd nodejs
npm install
npm run build       # gera em dist/
npm run typecheck   # tsc --noEmit
npm test            # vitest

# Go
cd go
go mod tidy
go build ./...
go test ./...
```

### Layout do projeto

```
afi-sdk/
├── nodejs/          ── @afi-run/sdk (TypeScript)
│   ├── src/
│   │   ├── client.ts, builder.ts          ── client público + builder de cotação
│   │   ├── token.ts, multicall.ts         ── reads ERC-20 + Multicall3
│   │   ├── swap.ts, quoter.ts             ── pipelines de swap + cotação
│   │   ├── address.ts, slippage.ts        ── helpers de DX
│   │   ├── serialize.ts, explorer.ts      ── helpers de JSON + URL
│   │   ├── errors.ts, types.ts            ── classes de erro + tipos públicos
│   │   ├── constants.ts, utils.ts         ── ABIs, endereços, unidades
│   │   └── index.ts                       ── exports públicos
│   └── src/__tests__/                     ── 159 testes unitários
├── go/              ── github.com/afi-run/sdk/go
│   ├── client.go, options.go              ── client público + functional options
│   ├── token.go, multicall.go             ── reads ERC-20 + Multicall3
│   ├── swap.go, quoter.go                 ── pipelines de swap + cotação
│   ├── address.go, slippage.go            ── helpers de DX
│   ├── serialize.go, explorer.go          ── helpers de JSON + URL
│   ├── errors.go, types.go                ── tipo de erro + tipos públicos
│   └── *_test.go                          ── 159 testes unitários
└── examples/        ── exemplos ponta-a-ponta executáveis
```

### Git hooks — `pre-push` espelha o CI localmente

O repo já vem com um hook de pre-push que roda exatamente os mesmos gates do CI
antes de qualquer `git push`, então você pega problemas em segundos em vez de
esperar PR verde.

```bash
# Instalação única (define core.hooksPath = scripts/git-hooks)
bash scripts/install-hooks.sh
```

No `git push`, o hook detecta quais subprojetos mudaram e roda:

| Subprojeto | Etapas |
|------------|--------|
| **Node.js** (quando `nodejs/` mudou) | `typecheck` · `vitest run --coverage` (≥95% stmts/lines/fns, ≥90% branches) · `npm audit --audit-level=high` |
| **Go** (quando `go/` ou `examples/go/` mudou) | `go vet` · `make test-coverage` (≥95%) · `go build` exemplos · `govulncheck` (instala se necessário) |

Atalhos:

```bash
SKIP_PRE_PUSH=1 git push      # pular dessa vez
PRE_PUSH_ALL=1  git push      # ignorar detecção de paths, rodar tudo
git config --unset core.hooksPath   # remover o hook completamente
```

### Estratégia de testes

O SDK é **totalmente testado em unit** sem dependências externas: chamadas RPC
e HTTP são mockadas. Rode com `npm test` ou `go test ./...`.

Em seus próprios testes, mocke a fronteira `AfiClient` / `*afi.Client` — o SDK
já confia que o RPC retorna respostas válidas, e esse é o ponto mais limpo para
suas fixtures.

---

## Licença

MIT © contribuidores do AFI Run. Veja [LICENSE](./LICENSE) para detalhes.
