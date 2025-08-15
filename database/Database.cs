using System.Data;
using System.Text.Json;
using local_gpss.models;
using local_gpss.utils;
using Microsoft.Data.Sqlite;
using PKHeX.Core;

namespace local_gpss.database;

public class Database
{
    private readonly SqliteConnection _connection;

    private Database()
    {
        string dbPath = Path.Combine(local_gpss.utils.Helpers.GetDataDirectory(), "gpss.db");
        _connection = new SqliteConnection($"Data Source={dbPath};foreign keys=true;");
        _connection.Open();

        Migrate();
    }

    public static Database? Instance { get; } = new();
    
    #region Pokemon Functions
    public dynamic? CheckIfPokemonExists(string base64, bool returnId = false)
    {
        var cmd = _connection.CreateCommand();
        cmd.CommandText = $"SELECT {(returnId ? "id" : "download_code")} FROM pokemon WHERE base_64 = @base64";
        cmd.Parameters.AddWithValue("@base64", base64);

        using var reader = cmd.ExecuteReader();
        if (!reader.Read())
        {
            return null;
        }

        return returnId ? reader.GetInt64(0) : reader.GetString(0);
    }
    
    public long InsertPokemon(string base64, bool legal, string code, string generation)
    {
        var cmd = _connection.CreateCommand();
        cmd.CommandText =
            @"INSERT INTO pokemon (upload_datetime, download_code, download_count, generation, legal, base_64) VALUES (@upload_datetime, @download_code, @download_count, @generation, @legal, @base_64); SELECT last_insert_rowid();";
        cmd.Parameters.AddWithValue("@upload_datetime", DateTime.Now);
        cmd.Parameters.AddWithValue("@download_code", code);
        cmd.Parameters.AddWithValue("@download_count", 0);
        cmd.Parameters.AddWithValue("@generation", generation);
        cmd.Parameters.AddWithValue("@legal", legal);
        cmd.Parameters.AddWithValue("@base_64", base64);
        
        return (long)cmd.ExecuteScalar();
    }
    #endregion
    
    #region Bundle Functions

    public string? CheckIfBundleExists(List<long> pokemonIds)
    {
        var valuesStr = string.Join(" UNION ALL ", pokemonIds.Select(id => $"SELECT {id} AS pokemon_id"));
        
        var cmd = _connection.CreateCommand();
        cmd.CommandText = $"""
                               WITH input_pokemon_ids AS (
                                   {valuesStr}
                               ), 
                               bundles_with_matching_pokemon AS (
                                   SELECT bundle_id
                                   FROM bundle_pokemon
                                   WHERE pokemon_id IN (SELECT pokemon_id FROM input_pokemon_ids)
                                   GROUP BY bundle_id
                                   HAVING COUNT(DISTINCT pokemon_id) = (SELECT COUNT(*) FROM input_pokemon_ids)
                               )
                               SELECT b.download_code
                               FROM bundles_with_matching_pokemon bp
                               JOIN bundle b ON bp.bundle_id = b.id
                               LIMIT 1
                           """;
        var reader = cmd.ExecuteReader();
        
        if (reader.Read())
        {
            return reader.IsDBNull(0) ? null : reader.GetString(0);
        }
        return null;
    }

    
    public void InsertBundle(bool legal, string code, string minGen, string maxGen, List<long> ids)
    {
        var cmd = _connection.CreateCommand();
        cmd.CommandText =
            @"INSERT INTO bundle (upload_datetime, download_code, download_count, legal, min_gen, max_gen) VALUES (@upload_datetime, @download_code, @download_count, @legal, @min_gen, @max_gen); SELECT last_insert_rowid();";
        cmd.Parameters.AddWithValue("@upload_datetime", DateTime.Now);
        cmd.Parameters.AddWithValue("@download_code", code);
        cmd.Parameters.AddWithValue("@download_count", 0);
        cmd.Parameters.AddWithValue("@legal", legal);
        cmd.Parameters.AddWithValue("@min_gen", minGen);
        cmd.Parameters.AddWithValue("@max_gen", maxGen);
        
        var bundleId = (long)cmd.ExecuteScalar();
        
        // Now to loop through and do a mass insert
        cmd.Parameters.Clear();

        cmd.CommandText = "INSERT INTO bundle_pokemon (pokemon_id, bundle_id) VALUES\n";
        for (var i = 0; i < ids.Count; i++)
        {
            cmd.CommandText += $"({ids[i]}, {bundleId})";
            if (i < ids.Count - 1)
            {
                cmd.CommandText += ",\n";
            }
            else
            {
                cmd.CommandText += ";";
            }
            
        }
        
        cmd.ExecuteNonQuery();
    }
    #endregion
    
    #region Generic Functions
    public bool CodeExists(string table, string code)
    {
        var cmd = _connection.CreateCommand();
        cmd.CommandText = $"SELECT EXISTS(SELECT 1 FROM {table} WHERE download_code = @code)";
        cmd.Parameters.AddWithValue("@code", code);

        return (Int64)cmd.ExecuteScalar() == 1;
    }

    public void IncrementDownload(string table, string code)
    {
        var cmd = _connection.CreateCommand();
        cmd.CommandText = $"UPDATE {table} SET download_count = download_count + 1 WHERE download_code = @code";
        cmd.Parameters.AddWithValue("@code", code);
        cmd.ExecuteNonQuery();
        if (table == "bundle")
        {
            cmd.CommandText = "UPDATE pokemon SET download_count = download_count + 1 WHERE id in (SELECT pokemon_id from bundle_pokemon where bundle_id = (select id from bundle where download_code = @code))";
            cmd.ExecuteNonQuery();
        }
        
    }
    
    public int Count(string table, Search? search = null)
    {
        var cmd = _connection.CreateCommand();
        var sql = GenerateBaseSelectSql(table, true, search);

        cmd.CommandText = sql;
        var reader = cmd.ExecuteReader();
        reader.Read();

        return reader.GetInt32(0);
    }


    public List<T> List<T>(string table, int page = 1, int pageSize = 30, Search? search = null)
    {
        var cmd = _connection.CreateCommand();
        var sql = GenerateBaseSelectSql(table, false, search);

        sql += "LIMIT " + pageSize;
        if (page > 1) sql += " OFFSET " + page * pageSize;

        cmd.CommandText = sql;
        using var reader = cmd.ExecuteReader();


        var items = new List<T>();
        var buffer1 = new List<GpssBundlePokemon>();
        var buffer2 = new List<string>();
        var buffer3 = new Dictionary<string, dynamic>();
        var currentDc = "";
        while (reader.Read())
        {
            if (table == "pokemon")
            {
                items.Add((T)Activator.CreateInstance(typeof(T), reader));
            } else {
                var dc = reader.GetString(reader.GetOrdinal("download_code"));
                if (currentDc == "")
                {
                    // get the current download code for the bundle
                    currentDc = dc;
                }
                else if (dc != currentDc)
                {
                    // Lists are pass by reference, this is dumb, it means I essentially have to duplicate the list
                    // to be able to clear the buffer so that the values actually get populated >_<
                    items.Add((T)Activator.CreateInstance(typeof(T), new List<GpssBundlePokemon>(buffer1), new List<string>(buffer2), buffer3));
                    buffer1.Clear();
                    buffer2.Clear();
                    buffer3.Clear();
                    currentDc = dc;
                }
                
                buffer1.Add(new GpssBundlePokemon()
                {
                    Generation = reader.GetString(reader.GetOrdinal("pg")),
                    Legal = reader.GetBoolean(reader.GetOrdinal("legality")),
                    Base64 = reader.GetString(reader.GetOrdinal("base_64")),
                });
                buffer2.Add(reader.GetString(reader.GetOrdinal("pdc")));
                if (!buffer3.Keys.Contains("download_count"))
                {
                    buffer3.Add("download_count", reader.GetInt64(reader.GetOrdinal("download_count")));
                    buffer3.Add("download_code", reader.GetString(reader.GetOrdinal("download_code")));
                    buffer3.Add("min_gen", reader.GetString(reader.GetOrdinal("min_gen")));
                    buffer3.Add("max_gen", reader.GetString(reader.GetOrdinal("max_gen")));
                    buffer3.Add("legal", reader.GetBoolean(reader.GetOrdinal("legal")));
                }
            }
        }
        
        if (table == "bundle" && buffer1.Count > 0)
        {
            items.Add((T)Activator.CreateInstance(typeof(T), new List<GpssBundlePokemon>(buffer1), new List<string>(buffer2), buffer3));
            buffer1.Clear();
            buffer2.Clear();
            buffer3.Clear();
        }

        return items;
    }
    #endregion

    private string GenerateBaseSelectSql(string table, bool count, Search? search = null)
    {
        var sql = $"SELECT {(count ? "COUNT(*)" : $"{(table == "pokemon" ? "*" : "bundle.*, pokemon.download_code as pdc, pokemon.generation as pg, pokemon.base_64, pokemon.legal as legality")}")} FROM {table} ";

        if (!count && table == "bundle")
        {
                sql += "INNER JOIN bundle_pokemon ON bundle.id = bundle_pokemon.bundle_id INNER JOIN pokemon ON bundle_pokemon.pokemon_id = pokemon.id ";
        }
        if (search.HasValue)
        {
            var needsAnd = false;
            if (search.Value.Generations != null)
            {
                needsAnd = true;
                // This kind of constructing queries is insecure as all hell. This is just another reminder that this
                // should server ABSOLUTELY NEVER be exposed to the public, this is for private internal usage only.
                // Want to make a public one? Write your own or run the risk of getting pwn'd.
                // I would use named parameters, but it doesn't support lists >_<
                if (table == "pokemon")
                {
                    sql += $"WHERE generation IN ('{string.Join("','", search.Value.Generations)}') ";
                }
                else
                {
                    var gens = string.Join("','", search.Value.Generations);
                    sql += $"WHERE min_gen IN ('{gens}') AND max_gen IN ('{gens}') ";
                }
            }

            if (search.Value.LegalOnly && needsAnd) sql += "AND " + (table == "bundle" ? " bundle.legal == 1 " : " legal == 1 ");
            else if (search.Value.LegalOnly) sql += (table == "bundle" ? "WHERE bundle.legal == 1 " : "WHERE legal == 1 ");
            
            
            if (search.Value.SortField != "")
            {
                sql += " ORDER BY " + search.Value.SortField + " " + (!search.Value.SortDirection ? " DESC " : " ASC ");
            }
        }

        return sql;
    }


    private void Migrate()
    {
        var cmd = _connection.CreateCommand();

        cmd.CommandText =
            """
            CREATE TABLE IF NOT EXISTS pokemon (
                id INTEGER PRIMARY KEY,
                upload_datetime DATETIME NOT NULL,
                download_code TEXT UNIQUE,
                download_count INTEGER,
                generation TEXT NOT NULL,
                legal BOOLEAN NOT NULL,
                base_64 TEXT NOT NULL
            );
            """;
        cmd.ExecuteNonQuery();

        cmd.CommandText =
            """
            CREATE TABLE IF NOT EXISTS bundle (
                id INTEGER PRIMARY KEY,
                download_code TEXT UNIQUE,
                upload_datetime DATETIME NOT NULL,
                download_count INTEGER,
                legal BOOLEAN NOT NULL,
                min_gen TEXT NOT NULL,
                max_gen TEXT NOT NULL
            )
            """;
        cmd.ExecuteNonQuery();

        cmd.CommandText =
            """
            CREATE TABLE IF NOT EXISTS bundle_pokemon (
                id INTEGER PRIMARY KEY,
                pokemon_id INTEGER,
                bundle_id INTEGER,
                FOREIGN KEY (pokemon_id) REFERENCES pokemon(id) ON DELETE CASCADE,
                FOREIGN KEY (bundle_id) REFERENCES bundle(id) ON DELETE CASCADE
            )
            """;
        cmd.ExecuteNonQuery();
    }
}