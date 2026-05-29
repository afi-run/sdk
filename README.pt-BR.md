# AFI SDK

SDK para troca de tokens na rede Base via [Protocolo AFI](https://afi.run).

Disponível em **Node.js (TypeScript)** e **Go**.

---

## Instalação

```bash
# Go — instalar direto do GitHub
go get github.com/afi-run/sdk/go

# Node.js — clonar e instalar pelo caminho local
git clone https://github.com/afi-run/sdk.git
npm install ./sdk/nodejs
```

> **Node.js:** o npm não suporta instalação de subdiretório do GitHub.
> Quando o pacote for publicado no npm, a instalação será simplesmente `npm install @afi-run/sdk`.

---

## Início rápido

```typescript
// Node.js — somente leitura primeiro, conectar assinante depois
import { AfiClient, formatUnits } from "@afi-run/sdk"

const client = new AfiClient({ rpcUrl: "https://rpc.ankr.com/base/SUA_CHAVE" })

const quote = await client.getQuote({
  tokenIn:  "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC
  tokenOut: "0x4200000000000000000000000000000000000006", // WETH
  amountIn: "1000",  // valor legível — sem raw wei
  slippage: 0.5,
})

console.log(`Você recebe: ~${quote.amountOut} WETH`)
console.log(`Mínimo garantido: ${quote.minOut} WETH`)

// Conectar assinante quando estiver pronto
client.connect("0xSUA_CHAVE_PRIVADA")
const result = await client.executeSwap(quote)
console.log("Tx:", result.txHash)
console.log("Recebido:", formatUnits(result.amountOut, 18), "WETH")
```

```go
// Go
client, _ := afi.NewClient(afi.Config{RPCURL: "https://rpc.ankr.com/base/SUA_CHAVE"})
defer client.Close()

quote, _ := client.GetQuote(ctx, afi.SwapParams{
    TokenIn:  common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
    TokenOut: afi.WETH,
    AmountIn: "1000",
    Slippage: 0.5,
})
fmt.Printf("Você recebe: ~%s WETH\n", quote.AmountOut)

client.Connect("SUA_CHAVE_PRIVADA")
result, _ := client.ExecuteSwap(ctx, quote)
fmt.Println("Recebido:", afi.FormatUnits(result.AmountOut, 18), "WETH")
```

---

## Referência da API

### Métodos somente leitura — sem chave privada necessária

#### `getTokens()` / `GetTokens(ctx)` — listar tokens disponíveis

```typescript
const tokens = await client.getTokens()
// Token[] — tokens ativos na rede Base

// Estrutura de Token:
// {
//   address:  "0x833589..."
//   symbol:   "USDC"
//   decimals: 6
//   active:   true
// }
```

Consulta `GET https://rpc.afi.run/info` e retorna os tokens suportados na Base.
Use isso na inicialização para permitir que o usuário escolha entre endereços válidos.
Não requer chave privada nem interação com a blockchain.

```go
// Go
tokens, err := client.GetTokens(ctx)
for _, t := range tokens {
    fmt.Printf("%s → %s\n", t.Symbol, t.Address.Hex())
}
```

---

#### `getFeeBps()` / `GetFeeBps(ctx)` — ler taxa atual do protocolo

```typescript
const feeBps = await client.getFeeBps()
// ex: 35 → 0.35%
```

Lê `feeBps` diretamente do contrato AFI. A taxa pode mudar e já está incluída
em todo objeto `Quote` retornado por `getQuote()`.

---

#### `getQuote(params)` / `GetQuote(ctx, params)` — obter cotação de preço

```typescript
const quote = await client.getQuote({
  tokenIn:  "0x...",   // endereço do token de entrada
  tokenOut: "0x...",   // endereço do token de saída
  amountIn: "1000",    // valor legível — sem raw wei
  slippage: 0.5,       // percentual, ex: 0.5 = 0.5%
})
```

**Somente leitura** — nenhuma transação é enviada. Seguro para chamar com frequência
para exibir preços em tempo real. Internamente:

1. Lê `decimals()` do contrato do token de entrada
2. Lê `feeBps()` do contrato AFI (em tempo real — a taxa pode mudar)
3. Chama `POST https://rpc.afi.run/quoter` com sua URL RPC

**Campos de `SwapParams`:**

| Campo | Tipo | Descrição |
|---|---|---|
| `tokenIn` | `string` / `Address` | Endereço do token de entrada |
| `tokenOut` | `string` / `Address` | Endereço do token de saída |
| `amountIn` | `string` | Valor legível, ex: `"1000"` ou `"0.5"` |
| `slippage` | `number` | Tolerância de slippage em percentual, ex: `0.5` |
| `maxHops` | `number?` | Número máximo de saltos na rota (opcional) |

**Campos de `Quote`:**

| Campo | Tipo | Descrição |
|---|---|---|
| `tokenIn` | `string` | Endereço do token de entrada |
| `tokenOut` | `string` | Endereço do token de saída |
| `amountIn` | `string` | Valor de entrada legível |
| `amountOut` | `string` | Saída estimada legível |
| `minOut` | `string` | Saída mínima legível após slippage |
| `amountInWei` | `bigint` | Entrada exata em Wei — use para `approve()` |
| `amountOutWei` | `bigint` | Saída estimada em Wei |
| `minOutWei` | `bigint` | Saída mínima em Wei — aplicada no contrato |
| `steps` | `Hex` | Rota codificada passada para `Afi.swap()` — não modificar |
| `path` | `Address[]` | Endereços dos tokens no caminho da rota |
| `hops` | `Hop[]` | Detalhamento por salto da rota |
| `slippage` | `number` | Percentual de slippage aplicado |
| `feeBps` | `number` | Taxa do protocolo no momento da cotação (basis points) |
| `tokenInPrice` | `string` | Preço de `tokenIn` denominado em `tokenOut` (taxa de câmbio) |
| `tokenOutPrice` | `string` | Preço de `tokenOut` denominado em `tokenIn` (taxa de câmbio) |

**Campos de `Hop`:**

| Campo | Tipo | Descrição |
|---|---|---|
| `tokenIn` | `string` | Token de entrada deste salto |
| `tokenOut` | `string` | Token de saída deste salto |
| `amountIn` | `string` | Entrada legível deste salto |
| `amountOut` | `string` | Saída legível deste salto |
| `minOut` | `string` | Saída mínima legível deste salto |
| `amountInWei` | `bigint` | Entrada deste salto em Wei |
| `amountOutWei` | `bigint` | Saída deste salto em Wei |
| `minOutWei` | `bigint` | Saída mínima deste salto em Wei |
| `tokenInPrice` | `string` | Preço do token de entrada deste salto denominado no token de saída |
| `tokenOutPrice` | `string` | Preço do token de saída deste salto denominado no token de entrada |
| `slippage` | `number` | Slippage aplicado a este salto |
| `type` | `string` | Identificador de tipo do salto |
| `kind` | `string` | Identificador de subtipo do salto |
| `routeId` | `number` | Identificador da rota |
| `weight` | `number` | Peso deste salto na rota geral |

**Importante:** `minOutWei` nunca é 0. O SDK rejeita cotações com saída mínima zero.

---

### Métodos de assinante — exigem `connect(privateKey)` primeiro

#### `connect(privateKey)` / `Connect(privateKey)` — vincular um assinante

```typescript
// Node.js — retorna `this` para encadeamento
client.connect("0xSUA_CHAVE_PRIVADA")

// ou passar na construção
const client = new AfiClient({ rpcUrl: "...", privateKey: "0x..." })
```

```go
// Go — retorna error
err := client.Connect("SUA_CHAVE_PRIVADA")
```

Vincula uma chave privada como assinante de transações. Todos os métodos somente leitura
funcionam sem chamar isso. Apenas `approve()`, `simulate()`, `submitSwap()`,
`executeSwap()` e `swap()` exigem um assinante conectado.

---

#### `approve(tokenIn, amountWei)` / `Approve(ctx, token, amountWei)` — aprovar apenas

```typescript
const pending = await client.approve(tokenIn, quote.amountInWei)
// Retorna null se o allowance já era suficiente (sem tx enviada)

if (pending) {
  console.log("Tx de aprovação:", pending.txHash)
  const receipt = await pending.wait()
  console.log("Confirmado no bloco:", receipt.blockNumber)
}
```

```go
approval, err := client.Approve(ctx, usdc, quote.AmountInWei)
if approval != nil {
    fmt.Println("Aprovação:", approval.TxHash)
    receipt, _ := approval.Wait(ctx)
    fmt.Printf("Confirmado no bloco %d\n", receipt.BlockNumber)
}
```

Aprova exatamente `amountWei` — sem excesso — para o contrato AFI. Retorna um
`PendingTx` (com `txHash` imediato) ou `null` se nenhuma aprovação foi necessária.

`executeSwap()` chama isso automaticamente. Use diretamente apenas se seu app precisar
de dois prompts de carteira separados (aprovar e depois trocar).

**Detalhes de segurança:**
- Verifica allowance on-chain primeiro — ignora se já suficiente
- Reseta para 0 antes de re-aprovar para tokens estilo USDT que exigem isso
- Reverifica allowance on-chain após a tx de aprovação confirmar

**Campos de `PendingTx`:**

| Campo | Tipo | Descrição |
|---|---|---|
| `txHash` | `string` | Hash da transação — disponível imediatamente |
| `wait()` | `() => Promise<TxReceipt>` | Aguardar confirmação on-chain |

**Campos de `TxReceipt`:**

| Campo | Tipo | Descrição |
|---|---|---|
| `blockNumber` | `bigint` | Bloco onde a transação foi confirmada |
| `gasUsed` | `bigint` | Gas consumido pela transação |

---

#### `simulate(quote, log?)` / `Simulate(ctx, quote, log?)` — simulação do swap

```typescript
const ok = await client.simulate(quote, console.error)
// Retorna true se o swap seria bem-sucedido, false se reverteria
if (!ok) return
```

```go
ok, err := client.Simulate(ctx, quote, func(r string) { fmt.Println("Falhou:", r) })
if !ok { return }
```

Executa uma simulação `eth_call` contra o estado atual da chain. Retorna `true` se o
swap seria bem-sucedido, `false` se reverteria. O callback opcional `log` recebe a
string de motivo de revert quando a simulação falha.

`executeSwap()` chama isso automaticamente antes de enviar qualquer transação.

---

#### `submitSwap(quote)` / `SubmitSwap(ctx, quote)` — enviar sem aguardar

```typescript
const pending = await client.submitSwap(quote)
console.log("Tx do swap:", pending.txHash)   // imediato

const result = await pending.wait()
console.log("Recebido:", formatUnits(result.amountOut, 18), "WETH")
```

```go
pending, err := client.SubmitSwap(ctx, quote)
fmt.Println("Swap:", pending.TxHash)

result, err := pending.Wait(ctx)
fmt.Println("Recebido:", afi.FormatUnits(result.AmountOut, 18), "WETH")
```

Envia a transação de swap e retorna um `PendingSwap` imediatamente — sem aguardar
confirmação. Chame `wait()` no objeto retornado para bloquear até o swap ser minerado.

**Campos de `PendingSwap`:**

| Campo | Tipo | Descrição |
|---|---|---|
| `txHash` | `string` | Hash da transação — disponível imediatamente |
| `wait()` | `() => Promise<SwapResult>` | Aguardar confirmação e obter resultado |

---

#### `executeSwap(quote)` / `ExecuteSwap(ctx, quote)` — executar uma cotação prévia

```typescript
const result = await client.executeSwap(quote)
```

Recebe um `Quote` retornado por `getQuote()` e executa o fluxo completo:

```
1. assertBalance  — verifica saldo do tokenIn ≥ amountInWei
2. approve        — aprova exatamente amountInWei para o contrato AFI
                    (ignorado se o allowance já é suficiente)
3. simulate       — executa eth_call antes de enviar — lança erro se a tx reverteria
4. swap           — envia a transação com estimativa de gas × 1.2
5. parse evento   — aguarda recibo e lê valores reais do evento SwapExecuted
```

**Por que simular primeiro?** Se a troca reverteria (ex: preço passou do `minOut`),
`SimulationFailedError` é lançado antes de qualquer gas ser gasto.

---

#### `swap(params)` / `Swap(ctx, params)` — atalho completo: cotação + execução

```typescript
const result = await client.swap({
  tokenIn:  "0x...",
  tokenOut: "0x...",
  amountIn: "500",
  slippage: 1.0,
})
```

Equivalente a `const q = await getQuote(params); return executeSwap(q)`.

Use para bots e scripts. Para apps com interface de usuário, prefira o padrão
`getQuote` → exibir preço → `executeSwap` para que o usuário confirme antes.

**Campos de `SwapResult`:**

| Campo | Tipo | Descrição |
|---|---|---|
| `txHash` | `string` | Hash da transação |
| `blockNumber` | `bigint` | Bloco onde o swap foi confirmado |
| `amountIn` | `bigint` | Entrada real do evento on-chain `SwapExecuted` |
| `amountOut` | `bigint` | Saída real do evento on-chain `SwapExecuted` |
| `tokenIn` | `string` | Endereço do token de entrada |
| `tokenOut` | `string` | Endereço do token de saída |
| `gasUsed` | `bigint` | Gas consumido pela transação |

---

### Helpers de unidade

#### `parseUnits(amount, decimals)` / `formatUnits(amount, decimals)`

```typescript
import { parseUnits, formatUnits } from "@afi-run/sdk"

// Legível → raw wei (bigint)
parseUnits("1000", 6)   // 1000_000000n  (1000 USDC)
parseUnits("1.5", 6)    // 1_500000n
parseUnits("0.5", 18)   // 500000000000000000n  (0.5 WETH)

// Raw wei (bigint) → legível
formatUnits(1000_000000n, 6)          // "1000"
formatUnits(1_500000n, 6)             // "1.5"
formatUnits(500000000000000000n, 18)  // "0.5"
```

```go
// Go
wei, err := afi.ParseUnits("1000", 6)   // big.Int 1000_000000
str := afi.FormatUnits(wei, 6)          // "1000"
```

`amountIn` em `SwapParams` já é legível, então você só precisa de `formatUnits`
para exibir `SwapResult.amountOut` e outros valores Wei bigint.

---

## Fluxo em etapas

O fluxo em etapas fornece um `txHash` para cada etapa imediatamente, antes de aguardar
confirmação — útil quando você precisa mostrar progresso em uma interface de usuário.

```typescript
// Node.js — fluxo completo em etapas
const client = new AfiClient({ rpcUrl: "..." })
const quote = await client.getQuote({ tokenIn: USDC, tokenOut: WETH, amountIn: "1000", slippage: 0.5 })

client.connect("0x...")

// 1. Aprovar
const approval = await client.approve(quote.tokenIn, quote.amountInWei)
if (approval) {
  console.log(`Tx de aprovação: ${approval.txHash}`)
  const receipt = await approval.wait()
  console.log(`Aprovado no bloco ${receipt.blockNumber}`)
}

// 2. Simular
const ok = await client.simulate(quote, console.error)
if (!ok) return

// 3. Enviar
const pending = await client.submitSwap(quote)
console.log(`Tx do swap: ${pending.txHash}`)

// 4. Aguardar
const result = await pending.wait()
console.log(`Recebido: ${formatUnits(result.amountOut, 18)} WETH`)
```

```go
// Go — fluxo completo em etapas
client, _ := afi.NewClient(afi.Config{RPCURL: "..."})
quote, _ := client.GetQuote(ctx, afi.SwapParams{TokenIn: usdc, TokenOut: afi.WETH, AmountIn: "1000", Slippage: 0.5})

client.Connect("SUA_CHAVE")

// 1. Aprovar
approval, _ := client.Approve(ctx, usdc, quote.AmountInWei)
if approval != nil {
    fmt.Println("Aprovação:", approval.TxHash)
    receipt, _ := approval.Wait(ctx)
    fmt.Printf("Confirmado no bloco %d\n", receipt.BlockNumber)
}

// 2. Simular
ok, _ := client.Simulate(ctx, quote, func(r string) { fmt.Println("Falhou:", r) })
if !ok { return }

// 3. Enviar
pending, _ := client.SubmitSwap(ctx, quote)
fmt.Println("Swap:", pending.TxHash)

// 4. Aguardar
result, _ := pending.Wait(ctx)
fmt.Println("Recebido:", afi.FormatUnits(result.AmountOut, 18), "WETH")
```

---

## Aprovação: por que sempre exata?

O SDK aprova exatamente o valor da cotação — nunca mais. Isso significa:

- Mesmo que o contrato AFI fosse comprometido, um atacante só poderia gastar o
  valor que você já ia gastar naquele swap específico
- Se você trocar com frequência, uma tx de aprovação será enviada por troca
- `executeSwap()` ignora a aprovação se o allowance existente já for suficiente

---

## Garantias de segurança

| Risco | Como o SDK trata |
|---|---|
| Bypass de slippage | `minOut` sempre vem da API do quoter — nunca definido como 0 |
| Aprovação excessiva | Sempre aprova exatamente `amountInWei` da cotação |
| Tokens estilo USDT | Reseta allowance para 0 antes de re-aprovar se necessário |
| Tx que reverteria | Simulação `eth_call` roda antes de cada swap — falha rápido |
| Race condition (allow vs swap) | Allowance reverificado on-chain após confirmação |
| Subestimativa de gas | Gas estimado on-chain e multiplicado por 1.2 |
| ETH nativo passado | Não suportado — use WETH: `0x4200000000000000000000000000000000000006` |

---

## Tratamento de erros

### Node.js

```typescript
import {
  NoSignerError,
  InsufficientBalanceError,
  SimulationFailedError,
  QuoteError,
  ApprovalError,
  SwapRevertedError,
} from "@afi-run/sdk"

try {
  const result = await client.executeSwap(quote)
} catch (e) {
  if (e instanceof NoSignerError) {
    // Chamou método de assinante sem conectar chave privada
    console.log("Conecte um assinante primeiro: client.connect(privateKey)")

  } else if (e instanceof InsufficientBalanceError) {
    // Usuário não tem saldo suficiente do tokenIn
    console.log("Saldo:    ", e.balance)   // bigint, raw wei
    console.log("Necessário:", e.required) // bigint, raw wei
    console.log("Token:    ", e.token)     // endereço

  } else if (e instanceof SimulationFailedError) {
    // Swap reverteria — nenhuma tx foi enviada, sem gas gasto
    console.log("Motivo:", e.reason)       // string de revert decodificada
    console.log("Dados: ", e.revertData)   // bytes raw (opcional)

  } else if (e instanceof QuoteError) {
    // API de cotação retornou erro (ex: sem rota encontrada)
    console.log(e.message)

  } else if (e instanceof ApprovalError) {
    // Transação de aprovação do token falhou
    console.log(e.message)

  } else if (e instanceof SwapRevertedError) {
    // Transação de swap reverteu on-chain
    console.log("Motivo:", e.reason)
  }
}
```

### Go

```go
import "errors"

result, err := client.ExecuteSwap(ctx, quote)
if err != nil {
    var afiErr *afi.AfiError
    if errors.As(err, &afiErr) {
        switch afiErr.Code {
        case "NO_SIGNER":
            // Chamou método de assinante sem conectar chave privada
            fmt.Println("Conecte um assinante primeiro: client.Connect(privateKey)")
        case "INSUFFICIENT_BALANCE":
            fmt.Println("Saldo insuficiente")
        case "SIMULATION_FAILED":
            // Nenhuma tx foi enviada
            fmt.Println("Reverteria:", afiErr.Message)
        case "QUOTE_FAILED":
            fmt.Println("Sem rota encontrada:", afiErr.Message)
        case "APPROVAL_FAILED":
            fmt.Println("Aprovação falhou:", afiErr.Message)
        case "SWAP_REVERTED":
            fmt.Println("Swap reverteu:", afiErr.Message)
        }
        return
    }
    log.Fatal(err) // erro inesperado
}
```

---

## Constantes

| Nome | Valor |
|---|---|
| Contrato AFI (Base) | `0xB8cC65321d169D55b93b4402D795701c6B308ce4` |
| WETH (Base) | `0x4200000000000000000000000000000000000006` |
| API Quoter | `https://rpc.afi.run/quoter` |
| API Info | `https://rpc.afi.run/info` |
| Chain ID | `8453` |

```typescript
import { AFI_ADDRESS, WETH } from "@afi-run/sdk"
```

```go
afi.AfiAddress // common.Address
afi.WETH       // common.Address
```

---

## Exemplos

### Node.js

| Arquivo | O que demonstra |
|---|---|
| `examples/nodejs/1-list-tokens.ts` | Listar tokens disponíveis |
| `examples/nodejs/2-get-quote.ts` | Obter e inspecionar cotação |
| `examples/nodejs/3-execute-swap.ts` | Cotação → revisão → execução (recomendado) |
| `examples/nodejs/4-full-flow.ts` | Atalho em uma chamada |
| `examples/nodejs/5-approve-only.ts` | Fluxo em etapas: aprovar, simular, enviar, aguardar |

```bash
cd nodejs
npm install
npx ts-node ../examples/nodejs/1-list-tokens.ts
```

### Go

| Diretório | O que demonstra |
|---|---|
| `examples/go/list-tokens/` | Listar tokens disponíveis |
| `examples/go/get-quote/` | Obter e inspecionar cotação |
| `examples/go/execute-swap/` | Cotação → revisão → execução (recomendado) |
| `examples/go/full-flow/` | Atalho em uma chamada |
| `examples/go/approve-only/` | Fluxo em etapas: aprovar, simular, enviar, aguardar |

```bash
cd examples/go
go mod tidy
go run ./list-tokens
go run ./get-quote
go run ./execute-swap
```

---

## Build a partir do código-fonte

```bash
# Node.js
cd nodejs
npm install
npm run build        # gera arquivos em dist/
npm run typecheck    # verificação de tipos apenas

# Go
cd go
go mod tidy
go build ./...
```
