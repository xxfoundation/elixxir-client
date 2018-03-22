package com.privategrity.bindingstest;

import android.support.v7.app.AppCompatActivity;
import android.os.Bundle;
import android.util.Log;

import java.math.BigInteger;

import bindings.Bindings;
import bindings.Message;
import bindings.Receiver;

public class MainActivity extends AppCompatActivity {

    final String TAG = "bindings_test";
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.i(TAG, "jsonSchema: "+ Bindings.getContactListJsonSchema());
        BigInteger userID = null;
        try {
            AndroidMessageReceiver receiver = new AndroidMessageReceiver();
            Bindings.initClient(new AndroidFakeRAMStorage(), "", receiver);
            // This uses the registration code for user 1
            // 10.0.2.2:50004 is the address of the last node of 5 nodes that I'm running locally on
            // my machine, accessed by the secret Android IP address for your local machine when running the Android emulator
            userID = new BigInteger(1, Bindings.register("2HOAAFKIVKEJ0", "Ben Smiley", "10.0.2.2:50000", 1));
            Log.i(TAG, "Registration returned UID: " + userID.toString());
            // Running login is only needed on the second and subsequent runs, if you stored the session data properly
            // This example doesn't store data properly, so we skip this, which breaks a lot of things
            String nick = Bindings.login(userID.toByteArray());

            // Try calling this and running the server with --noratchet if you're having trouble getting messages back from the server
            Bindings.disableRatchet();
            // You can actually set the nick for any user on the server. We'll control access to this better later.
            // Note: current code works wrong, always sets nick to David. We'll fix this.
            Bindings.setNick(userID.toByteArray(), "Angela Merkel");

            Bindings.send(new AndroidFakeMessage());
            while (!receiver.received) {
                Thread.sleep(10);
            }
            Log.i(TAG, "Received message: " + receiver.lastMessage);

            // Don't forget to log out when you're done
            Bindings.logout();
        } catch (Exception e) {
            Log.e(TAG, "Exception received when running bindings tests:", e);
        }


        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
    }
}
