using System.Text.Json.Serialization;

namespace GpssConsole.models;

public struct Config
{
    [JsonPropertyName("ip")] public string Ip { get; set; }
    [JsonPropertyName("port")] public int Port { get; set; }
}