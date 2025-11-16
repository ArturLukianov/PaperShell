/// PAPERSHELL_AGENT

function RegisterCommands(listenerType)
{

/// Commands Here

    let cmd_cat = ax.create_command("cat", "Read the specified file", "cat C:\\file.exe", "Task: read file");
    cmd_cat.addArgString("path", true);
    
    let cmd_cd = ax.create_command("cd", "Change directory", "cd C:\\Windows\\System32", "Task: change directory");
    cmd_cd.addArgString("path", true);
    
    let cmd_ls = ax.create_command("ls", "Get list of files and directories in path", "ls C:\\e", "Task: list directory");
    cmd_ls.addArgString("path", false);

    let cmd_run = ax.create_command("run", "Run executable and receive output", "run whoami /all", "Task: run executable");
    cmd_run.addArgString("executable", true);
    cmd_run.addArgString("args", false);

    if(listenerType == "PaperShellHTTP") {
        let commands_external = ax.create_commands_group("papershell", [cmd_cat, cmd_cd, cmd_ls, cmd_run] );

        return { commands_windows: commands_external }
    }
    return ax.create_commands_group("none",[]);
}

function GenerateUI(listenerType)
{
    // Пустая форма
    let container = form.create_container()

    let panel = form.create_panel()

    return {
        ui_panel: panel,
        ui_container: container
    }
}
