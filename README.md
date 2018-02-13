This repo will contain Privategrity clients for all platforms.

Preparation to Build
==

To build this client, you will need to get the Android SDK and NDK, and get and
initialize gomobile. I recommend getting the Android SDK and NDK using Android
Studio. Download it from https://developer.android.com/studio/index.html and
follow the instructions to install it. I recommend choosing the Standard setup
option.

After running setup, at the Android Studio splash screen, click
Configure-\>Settings towards the bottom. Navigate to Appearance &
Behavior-\>System Settings-\>Android SDK, then SDK Tools, then check NDK. Click
OK, follow the instructions, and before long you'll have an Android NDK
installation.

Export ANDROID\_HOME in your .profile or .bash\_profile, pointing to the
location of your Android SDK installation. Then, after confirming that the
environment variable change has taken effect:

 $ go get golang.org/x/mobile/cmd/gomobile
 
For Android support:
 $ gomobile init -ndk=/path/to/your/ndk/installation

On Linux, the NDK is in ~/Android/Sdk/ndk-bundle by default.

For iOS support: TODO.

In either case you will need to run $ gomobile init.

Building
==

To build the .aar for the client, cd to privategrity/client/android/client and
run this command:

$ gomobile bind -target=android gitlab.com/privategrity/client

Adding the .aar to the Android Studio project
==

In case you need to add another .aar to the Android Studio project, follow
these steps:

1. Go to File-\>New-\>New Module.
1. Scroll to and click on Import .JAR/.AAR Package.
1. Pick the .aar in the file chooser.
1. Click through the rest of the wizard.

In any case, this isn't a recommended course of action because there might be
some weirdness about gomobile generating more than one .aar.

