using System.CommandLine;
using System.CommandLine.Parsing;
using System.Text.Json;
using GpssConsole.utils;
using PKHeX.Core;

namespace GpssConsole;

class Program
{
    static int Main(string[] args)
    {
        Option<String> modeOption = new("--mode")
        {
            Description = "The mode to start GPSS Console for, legality | legalize",
            Required = true,
        };
        
        Option<String> pokemonBase64Option = new("--pokemon")
        {
            Description = "The Base64 encoded pokemon data to use for GPSS Console",
            Required = true,
        };
        
        Option<String> generationOption = new("--generation")
        {
            Description = "The generation of the pokemon to use for GPSS Console",
            Required = true,
        };
        
        Option<String> versionOption = new("--ver")
        {
            Description = "The version of the pokemon to use for GPSS Console",
        };
        
        RootCommand rootCommand = new("GPSS Console for Local GPSS");
        rootCommand.Options.Add(modeOption);
        rootCommand.Options.Add(pokemonBase64Option);
        rootCommand.Options.Add(generationOption);
        rootCommand.Options.Add(versionOption);
        rootCommand.Validators.Add(result =>
        {
            String? mode = result.GetValue(modeOption);
            if (mode is not "legalize" and  not "legality")
            {
                result.AddError("--mode must be legalize or legality");
            }
            
            String? version  = result.GetValue(versionOption);
            if (mode is "legalize" && version == null)
            {
                result.AddError("--ver is required for auto legalization");
            }
        });
        
        ParseResult parseResult = rootCommand.Parse(args);
        if (parseResult.Errors.Count == 0)
        {
            
            String mode = parseResult.GetRequiredValue(modeOption);
            String generation =  parseResult.GetRequiredValue(generationOption);
            String pokemon = parseResult.GetRequiredValue(pokemonBase64Option);
            // Get the entity context from the generation
            var ctx = Helpers.EntityContextFromString(generation);
            // Do the init
            Helpers.Init();

            if (mode == "legality")
            {
                var result = Pkhex.LegalityCheck(pokemon, ctx);

                Console.WriteLine(JsonSerializer.Serialize(result));
            }
            else
            {
                GameVersion version = Helpers.GameVersionFromString(parseResult.GetRequiredValue(versionOption));
                var result = Pkhex.Legalize(pokemon, ctx, version);
                Console.WriteLine(JsonSerializer.Serialize(result));
            }

            return 0;
        }
        
        foreach (ParseError parseError in parseResult.Errors)
        {
            Console.Error.WriteLine(parseError.Message);
        }
        return 1;
    }
}