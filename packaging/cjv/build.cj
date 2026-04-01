import std.env.*
import std.process.*

func isWindows(): Bool {
    (getVariable("OS") ?? "") == "Windows_NT"
}

func logHook(name: String): Unit {
    println("HOOK=${name}")
    for (arg in Process.current.arguments) {
        println("ARG=${arg}")
    }
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
    return 0
}

func stagePostBuild(): Int64 {
    return runInstaller("build")
}

func stagePreInstall(): Int64 {
    logHook("pre-install")
    return 0
}

func stagePostInstall(): Int64 {
    return runInstaller("install")
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
