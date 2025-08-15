using System.Dynamic;
using System.Net;
using System.Net.Sockets;
using System.Runtime.InteropServices;
using System.Security.Principal;
using System.Text.Json;
using local_gpss.database;
using local_gpss.models;
using PKHeX.Core;
using PKHeX.Core.AutoMod;

namespace local_gpss.utils;

public static class Helpers
{
    public static Random rand;

    public static string GetDataDirectory()
    {
        var dataDir = Environment.GetEnvironmentVariable("GPSS_DATA_DIR") ?? Directory.GetCurrentDirectory();
        if (!Directory.Exists(dataDir))
        {
            Directory.CreateDirectory(dataDir);
        }
        return dataDir;
    }

    public static Config? Init()
    {
        EncounterEvent.RefreshMGDB(string.Empty);
        RibbonStrings.ResetDictionary(GameInfo.Strings.ribbons);
        Legalizer.EnableEasterEggs = false;
        rand = new();
        
        // Load config 
        try
        {
            string configPath = Path.Combine(GetDataDirectory(), "local-gpss.json");
            string config = File.ReadAllText(configPath);
            return JsonSerializer.Deserialize<Config>(config);
        }
        catch (Exception e)
        {
            if (e is FileNotFoundException)
            {
                return null;
            }

            if (e is JsonException)
            {
                Console.WriteLine("Looks like something is wrong with your local-gpss.json configuration file!");
                Console.WriteLine("The following is error details if you want to share it with the developers");
                Console.WriteLine(e);
                Console.WriteLine("Luckily we can fix this by having you run through the configuration again!");
                return null;
            }
            
            Console.WriteLine("Uh Oh, something went wrong with reading the config for Local GPSS the error details are as followed:");
            Console.WriteLine(e);
            Console.WriteLine("Please provide these error details if you are asking for support.");
            Console.WriteLine("Press any key to exit..");
            Console.ReadKey();
            Environment.Exit(1);
        }
        
        return null;
    }

    public static EntityContext EntityContextFromString(string generation)
    {
        switch (generation)
        {
            case "1":
                return EntityContext.Gen1;
            case "2":
                return EntityContext.Gen2;
            case "3":
                return EntityContext.Gen3;
            case "4":
                return EntityContext.Gen4;
            case "5":
                return EntityContext.Gen5;
            case "6":
                return EntityContext.Gen6;
            case "7":
                return EntityContext.Gen7;
            case "8":
                return EntityContext.Gen8;
            case "9":
                return EntityContext.Gen9;
            case "BDSP":
                return EntityContext.Gen8b;
            case "PLA":
                return EntityContext.Gen8a;
            default:
                return EntityContext.None;
        }
    }

    public static GameVersion GameVersionFromString(string version)
    {
        if (!Enum.TryParse(version, out GameVersion gameVersion)) return GameVersion.Any;

        return gameVersion;
    }

    public static dynamic PokemonAndBase64FromForm(IFormFile pokemon, EntityContext context = EntityContext.None)
    {
        using var memoryStream = new MemoryStream();
        pokemon.CopyTo(memoryStream);

        return new
        {
            pokemon = EntityFormat.GetFromBytes(memoryStream.ToArray(), context),
            base64 = Convert.ToBase64String(memoryStream.ToArray())
        };
    }

    public static PKM? PokemonFromForm(IFormFile pokemon, EntityContext context = EntityContext.None)
    {
        using var memoryStream = new MemoryStream();
        pokemon.CopyTo(memoryStream);

        return EntityFormat.GetFromBytes(memoryStream.ToArray(), context);
    }

    // This essentially takes in the search format that the FlagBrew website would've looked for
    // and re-shapes it in a way that the SQL query can use.
    public static Search SearchTranslation(JsonElement query)
    {
        var search = new Search();

        var hasGens = query.TryGetProperty("generations", out var generations);
        if (hasGens)
        {
            List<string> gens = new();

            for (var i = 0; i < generations.GetArrayLength(); i++)
                switch (generations[i].GetString())
                {
                    case "LGPE":
                        gens.Add("7.1");
                        break;
                    case "BDSP":
                        gens.Add("8.2");
                        break;
                    case "PLA":
                        gens.Add("8.1");
                        break;
                    case null:
                        break;
                    default:
                        gens.Add(generations[i].GetString()!);
                        break;
                }

            search.Generations = gens;
        }

        var hasLegal = query.TryGetProperty("legal", out var legal);
        if (hasLegal) search.LegalOnly = legal.GetBoolean();

        var hasSortDirection = query.TryGetProperty("sort_direction", out var sort);
        if (hasSortDirection) search.SortDirection = sort.GetBoolean();

        var hasSortField = query.TryGetProperty("sort_field", out var sortField);
        if (hasSortField)
        {
            switch (sortField.GetString())
            {
                case "latest":
                    search.SortField = "upload_datetime";
                    break;
                case "popularity":
                    search.SortField = "download_count";
                    break;
                default:
                    search.SortField = "upload_datetime";
                    break;
            }
        }


        return search;
    }


    public static string GenerateDownloadCode(string table, int length = 10)
    {
        string code = "";
        while (true)
        {
            for (int i = 0; i < length; i++)
                code = String.Concat(code, rand.Next(10).ToString());


            // Now check to see if the code is in the database already and break if it isn't

            if (Database.Instance!.CodeExists(table, code))
            {
                continue;
            }

            break;
        }

        return code;
    }

    // Credit: https://stackoverflow.com/a/9956981
    public static bool DoesPropertyExist(dynamic obj, string name)
    {
        if (obj is ExpandoObject)
            return ((IDictionary<string, object>)obj).ContainsKey(name);

        return obj.GetType().GetProperty(name) != null;
    }

    public static Config NewIpNeeded(Config config)
    {
        Console.WriteLine("It appears the previous IP you had selected, is no longer usable (DHCP may have changed it)");
        Console.WriteLine("Don't worry, we'll have you select a new IP and then you'll be good to go!");
        config.Ip = ChooseIp();
        SaveConfig(config);
        Console.WriteLine("Config has been updated, thank you!");
        
        return config;
    }

    public static Config NewPortNeeded(Config config)
    {
        Console.WriteLine("It appears the previous port you had selected, is no longer usable (Might be in use already)");
        Console.WriteLine("You can either choose a new port, or restart Local GPSS after closing whatever is using it");
        Console.WriteLine("If you would like to use a new port, press y, otherwise this program will exit and you can relaunch it after you clear up the port");
        var key = Console.ReadKey();
        if (key.Key != ConsoleKey.Y)
        {
            Environment.Exit(3);
        }
        Console.WriteLine();
        
        config.Port = ChoosePort();
        SaveConfig(config);
        Console.WriteLine("Config has been updated, thank you!");
        return config;
    }
    
    public static Config FirstTime()
    {
        var config = new Config();
        string dbPath = Path.Combine(GetDataDirectory(), "gpss.db");
        bool isFirstTime = !File.Exists(dbPath);
        if (isFirstTime)
        {
            Console.WriteLine(
                "Howdy, it looks like this is the first time you're running Local GPSS");
        }
        else
        {
            Console.WriteLine("Howdy, it looks like this is the first time you're running a newer version of Local GPSS!");
        }
        
        Console.WriteLine(
            "You'll need to set some information first before you can use Local GPSS, don't worry this should be easy!");
        Console.WriteLine("First let's figure out what IP your computer is using.");
        config.Ip = ChooseIp();

        Console.WriteLine("Now you'll want to choose a port");
        config.Port = ChoosePort();

        SaveConfig(config);
        
        try
        {
            if (isFirstTime)
            {
                Console.WriteLine(
                    "Finally, it looks like this is your first time running Local GPSS (or running it from this directory) would you like to grab the gpss.db from the first release?");
                Console.WriteLine("If you do, please press y otherwise press any other key to continue");
                var key = Console.ReadKey();
                Console.WriteLine();
                if (key.Key == ConsoleKey.Y)
                {
                    Console.WriteLine("Okay we'll download gpss.db from GitHub now!");
                    // Download the gpss.db
                    using (var client = new HttpClient())
                    {
                        using (var s = client.GetStreamAsync(
                                   "https://github.com/FlagBrew/local-gpss/releases/download/v1.0.0/gpss.db"))
                        {
                            using (var fs = new FileStream(dbPath, FileMode.OpenOrCreate))
                            {
                                s.Result.CopyTo(fs);
                            }
                        }
                    }
                    Console.WriteLine("gpss.db has been successfully downloaded!");
                }
                else
                {
                    Console.WriteLine("No worries, you can always download it later if you want!");
                }
            }
            else
            {
                Console.WriteLine("Normally this is where you'd be prompted if you wanted to download gpss.db from the first release");
                Console.WriteLine("However, you already have a gpss.db so this likely isn't your first time, so we'll skip it!");
            }
        }
        catch (Exception e)
        {
            Console.WriteLine(
                "Uh oh, looks like we ran into a problem grabbing the gpss.db from GitHub");
            Console.WriteLine($"ERROR DETAILS: {e}");
            Console.WriteLine("If you are getting assistance, please provide the error details in your message");
            Console.WriteLine("Press any key to exit...");
            Console.ReadKey();
            Environment.Exit(1);
        }
 
        Console.WriteLine("Thank you for running through this configuration!");
        Console.WriteLine("We shall start the server now!");
        return config;
    }

    public static List<String> GetLocalIPs()
    {
        if (!System.Net.NetworkInformation.NetworkInterface.GetIsNetworkAvailable())
        {
            throw new Exception(
                "You are not connected to a network, you must have at-least a LAN (or Wi-Fi) connection to use this app.");
        }

        var host = Dns.GetHostEntry(Dns.GetHostName());
        var ips = new List<String>();
        foreach (var ip in host.AddressList)
        {
            if (ip.AddressFamily == AddressFamily.InterNetwork)
            {
                ips.Add(ip.ToString());
            }
        }

        if (ips.Count == 0)
        {
            throw new Exception("No network connections with an IPv4 address found in your system!");
        }

        return ips;
    }

    public static bool CanBindToPort(int port)
    {
        try
        {
            using (TcpListener listener = new TcpListener(IPAddress.Any, port))
            {
                listener.Start(); // Try to bind to the port
                listener.Stop(); // Release the port
                return true;
            }
        }
        catch (SocketException e)
        {
            return false; // Port is already in use or requires elevated permissions
        }
    }
    
    
    private static String ChooseIp()
    {
        string ip = "";
        try
        {
            var ips = GetLocalIPs();
            if (ips.Count == 1)
            {
                Console.WriteLine(
                    $"Good news! It appears your device only has a single IP address of {ips[0]} so we'll use that!");
                ip = ips[0];
            }
            else
            {
                Console.WriteLine(
                    $"Looks like you have more than one IP, I'll list them out for you, and then you'll specify (using the option number) which one you think it is.");
                for (int i = 0; i < ips.Count; i++)
                {
                    Console.WriteLine($"{i + 1}. {ips[i]}");
                }

                while (true)
                {
                    Console.WriteLine("Please choose the IP you wish to use.");
                    var choiceInput = Console.ReadLine();
                    if (!int.TryParse(choiceInput, out var choice) || choice < 1 || choice - 1 > ips.Count - 1)
                    {
                        Console.WriteLine("Invalid choice. Please try again.");
                        continue;
                    }

                    ip = ips[choice - 1];
                    break;
                }
            }
        }
        catch (Exception e)
        {
            Console.WriteLine(
                "Uh oh, looks like we ran into a problem getting your device's IP address, please review the error details below");
            Console.WriteLine($"ERROR DETAILS: {e}");
            Console.WriteLine("If you are getting assistance, please provide the error details in your message");
            Console.WriteLine("Press any key to exit...");
            Console.ReadKey();
            Environment.Exit(1);
        }

        return ip;
    }

    private static int ChoosePort()
    {
        int minPort = 1;
        if (RuntimeInformation.IsOSPlatform(OSPlatform.Linux) || RuntimeInformation.IsOSPlatform(OSPlatform.OSX))
        {
            var canUsePrivilegedPorts = CanBindToPort(80);
            Console.WriteLine(
                "Please note, that because you are on a Linux (or Mac) Operating System, by default you will not be able to use ports 1-1024 as a regular user");
            Console.WriteLine("You could if you have it configured to let your user use them");

            if (!canUsePrivilegedPorts)
            {
                Console.WriteLine(
                    "However, based on a quick check, it appears you cannot bind to those ports, so please use a port higher than 1024.");
                minPort = 1025;
            }
            else
            {
                Console.WriteLine(
                    "However, based on a quick check, it appears you can bind to those ports, so you may use of them, but please be careful.");
                Console.WriteLine();
            }
        }

        int port;

        while (true)
        {
            Console.WriteLine($"Please enter a port between {minPort} and 65535");
            var inputtedPort = Console.ReadLine();
            if (!int.TryParse(inputtedPort, out port)  || port < minPort || port > 65535)
            {
                Console.WriteLine("Invalid port. Please try again.");
                continue;
            }

            if (!CanBindToPort(port))
            {
                Console.WriteLine($"Cannot use port {port}, please choose another port");
                continue;
            }

            break;
        }

        return port;
    }



    public static bool IsRunningAsAdminOrRoot()
    {
        if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows))
        {
            using (WindowsIdentity identity = WindowsIdentity.GetCurrent())
            {
                WindowsPrincipal principal = new WindowsPrincipal(identity);
                return principal.IsInRole(WindowsBuiltInRole.Administrator);
            }
        }
        else
        {
            return geteuid() == 0; // On macOS/Linux, UID 0 is root
        }
    }

    private static void SaveConfig(Config config)
    {
        try
        {
            string json = JsonSerializer.Serialize(config);
            string configPath = Path.Combine(GetDataDirectory(), "local-gpss.json");
            File.WriteAllText(configPath, json);
        }
        catch (Exception e)
        {
            Console.WriteLine(
                "Uh oh, looks like we ran into a problem saving the config to disk, please review the error details below");
            Console.WriteLine($"ERROR DETAILS: {e}");
            Console.WriteLine("If you are getting assistance, please provide the error details in your message");
            Console.WriteLine("Press any key to exit...");
            Console.ReadKey();
            Environment.Exit(1);
        }

    }

    [DllImport("libc")]
    private static extern uint geteuid();
}