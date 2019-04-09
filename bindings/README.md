# Supporting Elixxir payments

### Protocol buffer crash course for Elixxir

#### Why use protocol buffers?

Protocol buffer definitions (written in a .proto file) can generate code for serializing and deserializing byte sequences in a huge variety of languages. This means that, if you decide to standardize your message format with a protocol buffer declaration, you can decode and encode the messages in any language without having to interact with the Go client library at all.  Just as important is the ability to define enumerations in protocol buffers. In particular, the message types that are currently used for the CUI and the command-line client are defined in the Types enum in `client/cmixproto/types.proto`. If you have any questions about the way that the data are serialized for any message type, you should read this file and its comments.

#### Generating protocol buffer code

To generate the code, use the `protoc` tool, requesting output in the target language of your choice. For Go:

`protoc --go_out=. types.proto`

For Java:

`protoc --java_out=/path/to/android/project/source/code types.proto`

You can download and install the protocol buffer compiler for your preferred language from [its downloads page](https://developers.google.com/protocol-buffers/docs/downloads).

#### Message types used to implement payments UI

The payments portion of the user interface should only need to register its listeners with the wallet. To get the wallet (currently there's only one), call `Bindings.getActiveWallet()`. You can listen to the wallet with `Bindings.getActiveWallet().Listen(...)`. When the client has received a message that the UI needs to respond to, it should listen to the wallet for messages of these types: `PAYMENT_INVOICE_UI`, `PAYMENT_RESPONSE`, and `PAYMENT_RECEIPT_UI`. See cmixproto/types.proto for some documentation about these file formats.

### What must client implementers do for the minimal payment implementation?

There are three parties that must participate to complete a payment: the payer, the payee, and the payment bots. Payer and payee are both normal, human users using some Elixxir client program, and the payment bots are automatic users that arbitrate exchanges of which tokens have value.
A payment happens like this: payee sends invoice with some payee-owned tokens that don't yet have value, payer receives invoice and decides to pay it, payer sends payment message to bots with payee-owned tokens and some tokens of his own that are worth the same and have value, bots locally see that the payer's tokens have value, bots store value in the payee's new tokens and destroy the payer's old tokens, bots reply to the payer with a message saying that the payment was successful, payer responds to the payee with a message asserting that they made the payment.

Under the hood, the client library updates a lot of state to perform the necessary cryptography to update the wallet to its correct state. At the time, though, the mechanism for rolling back failed transactions is relatively untested and needs some ironing out. In the meantime, if a transaction fails and further transactions are messed up, you should restart the whole server infrastructure--server, gateway, and payment bot--to reset the tokens that are available on the payment bot, and wipe the stored sessions of the clients to register with a fresh set of tokens.

In short, if you're implementing a client, your client must be able to do the following things to deliver the baseline payments experience:

- send an invoice to another user on the payee's client
- receive and display an incoming invoice on the payer's client
- pay the received invoice on the payer's client
- receive and display the payment bots' response on the payer's client
- receive and display the payer's receipt on the payee's client

How to do each of these things with the current payments API follows. Assume that `w` is a reference or pointer to the active wallet. Assume that `CmixProto` is an imported package with proto buffer code generated in Java. The code is written in Java-like pseudocode. Of course, you should structure your own code in the best way for your own application.

#### 0. Actually having tokens to spend

To actually have tokens to spend, you must mint the same tokens as are on the payment bot. Currently, the tokens are hard-coded. To do this, pass `true` to the last parameter of `Bindings.Register()`. Then, there will be tokens that are stored in the wallet that happen to be the same as the tokens that are stored on the payment bot (when the payment bot is run with `--mint`), and the client will be able to spend them.

#### 1. Send an invoice to another user on the payee's client

First, generate the invoice, then send it.

```java
public static void sendInvoice() throws Throwable {
    // Generate the invoice message: Request 500 tokens from the payer
    Message invoiceMessage = w.Invoice(payerId.bytes(), 500,
        "for creating a completely new flavor of ice cream");

    // Send the invoice message to the payer
    Bindings.send(invoiceMessage); 
}
```

#### 2. Receive and display an incoming invoice on the payer's client

During client startup, register a listener with the wallet. The wallet has a separate listener matching structure from the main switchboard, and you can use it to receive messages from the wallet rather than from the network.

```java
public static void setup() throws Throwable {
    Bindings.InitClient(...);
    Bindings.Register(...);

    // The wallet and listener data structure (switchboard) are both ready after Register.
    // On the other hand, the client begins receiving messages after Login. So, if
    // you want to receive all messages that the client receives, it's best to register
    // listeners between Register and Login.
    registerListeners();

    Bindings.Login(...);
}

public static void registerListeners() {
    // Listen for messages of type PAYMENT_INVOICE_UI originating from all users
    w.Listen(zeroId.bytes(), CmixProto.Type.PAYMENT_INVOICE_UI, new PaymentInvoiceListener());
    // and so on...
}
```

The message you'll get contains an invoice ID. You can use it to query the invoice's transaction for display. You should also store the invoice ID as it's the parameter for the Pay method, the next phase.

```java
public class PaymentInvoiceListener implements Bindings.Listener {
    @Override
    public void Hear(Bindings.Message msg, bool isHeardElsewhere) {
        // Keep this ID around somewhere for the next step
        byte[] invoiceID = msg.GetPayload();
        bindings.Transaction invoice = w.GetInboundRequests.Get(invoiceID);

        // Display the transaction somehow
        invoiceDisplay.setTime(invoice.timestamp);
        invoiceDisplay.setValue(invoice.value);
        invoiceDisplay.setMemo(invoice.memo);
        invoiceDisplay.show();
    }
}
```

#### 3. Pay the received invoice on the payer's client

When the payer approves the invoice's payment, call the Pay() method with the invoice ID. This will generate a message that the payer can send to the payment bot. The message contains the payee's tokens, which the payment bot will vest, and the payer's proof of ownership of tokens of equal value, which the payment will destroy to facilitate the exchange.

```java
public static void sendPayment() throws Throwable {
    Message msg = Bindings.pay(invoiceID);
    Bindings.send(msg);
}
```

#### 4. Receive and display the payment bots' response on the payer's client

When you register listeners, listen to the wallet for the type `PAYMENT_RESPONSE`.

```java
public static void registerListeners() {
    // ...
    w.Listen(zeroId.bytes(), CmixProto.Type.PAYMENT_RESPONSE, new PaymentResponseListener());
    // ...
}
```

The payment response is a serialized protocol buffer with a few fields. You should deserialize it using the protocol buffer code you generated.

```java
public class PaymentResponseListener implements Binding.Listener {
    @Override
    public void Hear(Bindings.Message msg, bool isHeardElsewhere) {
        // Parse the payment bot's response
        PaymentResponse response = PaymentResponse.parseFrom(msg.GetPayload());

        // Then, show it to the user somehow
        responseDisplay.setSuccess(response.getSuccess());
        responseDisplay.setText(response.getResponse());
        responseDisplay.show();
    }
}
```

The client automatically converts the payment bot's response to a receipt and sends it on to the payee.

#### 5. Receive and display the payer's receipt on the payee's client

When you register listeners, listen to the wallet for the type `PAYMENT_RECEIPT_UI`.

```java
public static void registerListeners() {
    // ...
    w.Listen(zeroId.bytes(), CmixProto.Type.PAYMENT_RECEIPT_UI, new PaymentReceiptListener());
    // ...
}
```

The payment receipt UI message is the ID of the original invoice that was paid. You can get the transaction itself from the CompletedOutboundPayments transaction list, then display the transaction information of the completed payment.

```java
public class PaymentReceiptListener implements Bindings.Listener {
    @Override
    public void Hear(bindings.Message msg, bool isHeardElsewhere) {
        // Get the relevant transaction
        bindings.Transaction completedTransaction = w.getCompletedOutboundTransaction(msg.GetPayload());

        // Show the receipt, including the time that the original invoice was sent.
        receiptDisplay.setTime(completedTransaction.time);
        receiptDisplay.setMemo(completedTransaction.memo);
        receiptDisplay.setValue(completedTransaction.value);
        receiptDisplay.show();
    }
}
```

### Cyclic group for registration JSON format: 

```json
{
  "gen": "5c7ff6b06f8f143fe8288433493e4769c4d988ace5be25a0e24809670716c613d7b0cee6932f8faa7c44d2cb24523da53fbe4f6ec3595892d1aa58c4328a06c46a15662e7eaa703a1decf8bbb2d05dbe2eb956c142a338661d10461c0d135472085057f3494309ffa73c611f78b32adbb5740c361c9f35be90997db2014e2ef5aa61782f52abeb8bd6432c4dd097bc5423b285dafb60dc364e8161f4a2a35aca3a10b1c4d203cc76a470a33afdcbdd92959859abd8b56e1725252d78eac66e71ba9ae3f1dd2487199874393cd4d832186800654760e1e34c09e4d155179f9ec0dc4473f996bdce6eed1cabed8b6f116f7ad9cf505df0f998e34ab27514b0ffe7",
  "prime": "9db6fb5951b66bb6fe1e140f1d2ce5502374161fd6538df1648218642f0b5c48c8f7a41aadfa187324b87674fa1822b00f1ecf8136943d7c55757264e5a1a44ffe012e9936e00c1d3e9310b01c7d179805d3058b2a9f4bb6f9716bfe6117c6b5b3cc4d9be341104ad4a80ad6c94e005f4b993e14f091eb51743bf33050c38de235567e1b34c3d6a5c0ceaa1a0f368213c3d19843d0b4b09dcb9fc72d39c8de41f1bf14d4bb4563ca28371621cad3324b6a2d392145bebfac748805236f5ca2fe92b871cd8f9c36d3292b5509ca8caa77a2adfc7bfd77dda6f71125a7456fea153e433256a2261c6a06ed3693797e7995fad5aabbcfbe3eda2741e375404ae25b",
  "primeQ": "f2c3119374ce76c9356990b465374a17f23f9ed35089bd969f61c6dde9998c1f"
}
```
