using System.Dynamic;
using PKHeX.Core;
using PKHeX.Core.AutoMod;

namespace GpssConsole.utils;

public static class Helpers
{
    public static void Init()
    {
        EncounterEvent.RefreshMGDB(string.Empty);
        Legalizer.EnableEasterEggs = false;
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
    
    public static PKM? PokemonFromBase64(String pokemon, EntityContext context = EntityContext.None)
    {
        try
        {
            var bytes = Convert.FromBase64String(pokemon);

            return EntityFormat.GetFromBytes(bytes, context);
        }
        catch (Exception)
        {
            return null;
        }
    }
}