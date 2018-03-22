package com.privategrity.bindingstest;

import bindings.Message;
import bindings.Receiver;

/**
 * Created by spencer on 3/21/18.
 */

public class AndroidMessageReceiver implements Receiver {
    public String lastMessage;
    public boolean received;

    @Override
    public void receive(Message message) {
        lastMessage = message.getPayload();
        received = true;
    }
}
