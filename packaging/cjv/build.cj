import std.convert.*
import std.env.*
import std.fs.*
import std.process.*
import std.time.*

let installMarkerDir = Path("./build-script-cache")
let installMarkerPath = installMarkerDir.join("install.marker")
let installMarkerMaxAgeMillis: Int64 = 10 * 60 * 1000

func isWindows(): Bool {
    (getVariable("OS") ?? "") == "Windows_NT"
}

func nowMillis(): Int64 {
    DateTime.now().toUnixTimeStamp().toMilliseconds()
}

func logHook(name: String): Unit {
    println("HOOK=${name}")
    for (arg in Process.current.arguments) {
        println("ARG=${arg}")
    }
}

func installMarkerExists(): Bool {
    try {
        let info = FileInfo(installMarkerPath)
        return info.isRegular() || info.isSymbolicLink()
    } catch (_: Exception) {
        return false
    }
}

func installMarkerDirExists(): Bool {
    try {
        let info = FileInfo(installMarkerDir)
        return info.isDirectory() || info.isSymbolicLink()
    } catch (_: Exception) {
        return false
    }
}

func writeInstallMarker(value: Int64): Unit {
    try {
        if (!installMarkerDirExists()) {
            Directory.create(installMarkerDir, recursive: true)
        }
        File.writeTo(installMarkerPath, value.toString().toArray())
    } catch (err: Exception) {
        eprintln("failed to update install marker ${installMarkerPath}")
        eprintln("${err}")
    }
}

func clearInstallMarker(): Unit {
    writeInstallMarker(0)
}

func installMarkerAgeMillis(): ?Int64 {
    if (!installMarkerExists()) {
        return None
    }

    try {
        let raw = String.fromUtf8(File.readFrom(installMarkerPath)).trimAscii()
        if (raw.isEmpty()) {
            return None
        }

        let startedAt = Int64.parse(raw)
        if (startedAt <= 0) {
            return None
        }
        return nowMillis() - startedAt
    } catch (err: Exception) {
        eprintln("failed to read install marker ${installMarkerPath}")
        eprintln("${err}")
        return None
    }
}

func cleanupStaleInstallMarker(): Unit {
    if (let Some(age) <- installMarkerAgeMillis() && age > installMarkerMaxAgeMillis) {
        clearInstallMarker()
    }
}

func shouldInstallRealBinary(): Bool {
    if (let Some(age) <- installMarkerAgeMillis() && age <= installMarkerMaxAgeMillis) {
        clearInstallMarker()
        return true
    }

    cleanupStaleInstallMarker()
    return false
}

func runInstaller(mode: String): Int64 {
    logHook("dispatch-${mode}")

    try {
        let workingDirectory = getWorkingDirectory()
        if (isWindows()) {
            return execute(
                "powershell",
                [
                    "-NoProfile",
                    "-ExecutionPolicy",
                    "Bypass",
                    "-File",
                    "./scripts/install-cjv.ps1",
                    "--mode",
                    mode,
                ],
                workingDirectory: workingDirectory,
            )
        }

        return execute(
            "sh",
            [
                "./scripts/install-cjv.sh",
                "--mode",
                mode,
            ],
            workingDirectory: workingDirectory,
        )
    } catch (err: Exception) {
        eprintln("failed to execute installer hook for ${mode}")
        eprintln("${err}")
        return 1
    }
}

func stagePreBuild(): Int64 {
    logHook("pre-build")
    cleanupStaleInstallMarker()
    return 0
}

func stagePostBuild(): Int64 {
    logHook("post-build")
    if (shouldInstallRealBinary()) {
        return runInstaller("build")
    }
    return 0
}

func stagePreInstall(): Int64 {
    logHook("pre-install")
    writeInstallMarker(nowMillis())
    return 0
}

func stagePostInstall(): Int64 {
    logHook("post-install")
    clearInstallMarker()
    return 0
}

main(): Int64 {
    if (Process.current.arguments.size == 0) {
        eprintln("build script hook name is missing")
        return 1
    }

    let hook = Process.current.arguments[0]
    if (hook == "pre-build") {
        return stagePreBuild()
    }
    if (hook == "post-build") {
        return stagePostBuild()
    }
    if (hook == "pre-install") {
        return stagePreInstall()
    }
    if (hook == "post-install") {
        return stagePostInstall()
    }

    logHook(hook)
    return 0
}
