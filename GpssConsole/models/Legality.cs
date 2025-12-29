using System.Text.Json.Serialization;
using PKHeX.Core;

namespace GpssConsole.models;

public struct LegalityCheckReport(LegalityAnalysis la)
{
    [JsonPropertyName("legal")] public bool Legal { get; set; } = la.Valid;
    [JsonPropertyName("report")] public string[] Report { get; set; } = la.Report().Split("\n");
}

public struct AutoLegalizationResult
{
    [JsonPropertyName("legal")] public bool Legal { get; set; }
    [JsonPropertyName("success")] public bool Success { get; set; } = false;
    [JsonPropertyName("ran")] public bool Ran { get; set; }
    [JsonPropertyName("report")] public string[] Report { get; set; }
    [JsonPropertyName("pokemon")] public string? PokemonBase64 { get; set; }

    public AutoLegalizationResult(LegalityAnalysis la, PKM? pokemon, bool ran)
    {
        Legal = la.Valid;
        Report = la.Report().Split("\n");
        Success = la.Valid;
        Ran = ran;

        if (pokemon == null) return;
        PokemonBase64 = Convert.ToBase64String(pokemon.SIZE_PARTY > pokemon.SIZE_STORED
            ? pokemon.DecryptedPartyData
            : pokemon.DecryptedBoxData);
    }
}