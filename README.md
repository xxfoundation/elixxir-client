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

After these steps are complete, run $ glide up to get all the other Go
dependencies.

Building
==

To build the .aar for the client, cd to privategrity/client/android/client and
run this command:

$ gomobile bind -target=android gitlab.com/privategrity/client/api

Adding the .aar to the Android Studio project
==

In case you need to add another .aar to the Android Studio project, follow
these steps:

1. Go to File-\>New-\>New Module.
1. Scroll to and click on Import .JAR/.AAR Package.
1. Pick the .aar in the file chooser.
1. Click through the rest of the wizard.

In any case, this isn't a recommended course of action because there might be
some weirdness about gomobile generating more than one .aar, and the .aar that
building generates has already been added to the project.

Project Structure
==

The top-level Go package is called "client". This package contains all of the
APIs that the mobile apps will be able to access. If the mobile apps shouldn't
be able to access some functions directly, put those functions in a different
package!

The package crypto will contain client cryptops, and the package comms will
contain client communications. When working tickets related to crypto or
communications, ask yourself whether the code you're writing would belong
better in the crypto or comms repositories instead.

Gomobile Usage and Caveats
==

Every exported symbol from the "client" package should be exported by the
gomobile bindings. Only the following data types are allowed to be exported:

- Signed integer and floating point types.

- String and boolean types.

- Byte slice types. Note that byte slices are passed by reference,
  and support mutation.

- Any function type all of whose parameters and results have
  supported types. Functions must return either no results,
  one result, or two results where the type of the second is
  the built-in 'error' type.

- Any interface type, all of whose exported methods have
  supported function types.

- Any struct type, all of whose exported methods have
  supported function types and all of whose exported fields
  have supported types.

So, exporting a cyclic Int _shouldn't_ work, and you should get an error if you
try to. But it's totally OK to have a cyclic Int in the "client" package, as
long as it's not exported, and it's totally OK to have a cyclic Int exported
from a package that "client" depends on.

See https://godoc.org/golang.org/x/mobile/cmd/gobind for other important usage
notes (avoiding reference cycles, calling Java or Objective-C from Go.)
