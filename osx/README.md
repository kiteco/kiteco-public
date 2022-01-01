Xcode
-----
You will need Xcode on your machine to build the Kite front-end.

You should be able to open the Xcode project at:

```sh
~/go/src/github.com/kiteco/kiteco/osx/Kite.xcodeproj
```

Open the project, and make sure that the `Team` under `Signing (Debug)` is the same for each of `Kite.xcodeproj`, `KiteHelper.xcodeproj`, and `KiteSidebar.xcodeproj` (you're fine just signing with your personal team). Then hit the `Run` button. You should now see the Kite icon on your menubar. (This will build the Go component of the client automatically.)

The build might fail if you have not built the electron sidebar. The development flow for working with the sidebar requires invoking a separate command to build the electron application before running the Xcode project. You might see this build error in Xcode:

```
Checking for electon/Kite.app...
Please run 'osx/build_electron.sh force' to build the sidebar application
Command /bin/sh failed with exit code 1
```

Follow the instructions (e.g run `osx/build_electron.sh force`), and build the Xcode project again. Note that if you are expecting updates to the electron application (e.g after a `git pull`), you must manually rebuild the application by running this command again. Otherwise, the application that is build by Xcode will use whatever electron bundle was previously built.

If you are working directly with the Electron sidebar, please see the developer instructions at https://github.com/kiteco/kiteco/tree/master/sidebar.

### Code signing libraries

The build expects committed libraries (Sparkle, Rollbar) to be already signed, and ready to be bundled directly into `Kite.app`.
In order to do this, reference https://github.com/sparkle-project/Sparkle/issues/1389#issuecomment-507950890.

```
codesign --verbose --force --deep -o runtime --sign "Developer ID Application" "$LOCATION/Sparkle.framework/Versions/A/Resources/AutoUpdate.app"
codesign --verbose --force -o runtime --sign "Developer ID Application" "$LOCATION/Sparkle.framework"
```
