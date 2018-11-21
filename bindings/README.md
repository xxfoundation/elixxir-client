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

After the payer's clients and payee's clients have gone through all of these steps, you'll have successfully processed a payment transaction.

