package com.privategrity.bindingstest;

import bindings.Storage;

/**
 * Created by spencer on 3/16/18.
 */

public class AndroidFakeRAMStorage implements Storage {
    byte[] storage;
    String location;
    // Obviously you would implement these differently to be able to save the session data
    @Override
    public String getLocation() {
        return location;
    }

    @Override
    public byte[] load() {
        return storage;
    }

    @Override
    public void save(byte[] bytes) throws Exception {
        storage = bytes;
    }

    @Override
    public void setLocation(String s) throws Exception {
        location = s;
    }
}
