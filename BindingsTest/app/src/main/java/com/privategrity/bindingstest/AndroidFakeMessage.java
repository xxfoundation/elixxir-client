package com.privategrity.bindingstest;

import java.math.BigInteger;

import bindings.Message;

/**
 * Created by spencer on 3/16/18.
 */

public class AndroidFakeMessage implements Message {
    @Override
    public String getPayload() {
        return "Hello";
    }

    @Override
    public byte[] getRecipient() {
        return new BigInteger("1").toByteArray();
    }

    @Override
    public byte[] getSender() {
        return new BigInteger("1").toByteArray();
    }
}
