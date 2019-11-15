package database

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "os"
    "strconv"
    "log"
)

// Database type
type Database struct {
    File string
}

// New Database constructor
func New(dbFile string) Database {
    db := Database{File: dbFile}
    return db
}

// newDatabase private method that creates a new database
func (db *Database) newDatabase() {
    dbFile, err := os.Create(db.File)
    if err != nil {
        panic(err)
    }
    dbFile.Close()
    statement := `
        create table karma (channel text, word text, karma integer);
        delete from karma;
        create table alias (channel text, alias text, word text);
        delete from alias;
        create table settings (channel text, setting text, value text);
        delete from settings;
        create table admins (channel text, user text);
        delete from admins;
        `
    db.runStatement(statement)
}

// Connect Returns "a connection" to the database
func (db *Database) Connect() {
    log.Print("Checking if database file already exists")
    if _, err := os.Stat(db.File); err == nil {
        log.Print("Database file already exists")
    } else if os.IsNotExist(err) {
        log.Print("Database does not exists, creating a new one")
        db.newDatabase()
    } else {
        panic(err)
    }
}

// runStatement runs a query into the db
func (db *Database) runStatement(statement string) {
    database, err := sql.Open("sqlite3", db.File)

    if err != nil {
        panic(err)
    }

    defer database.Close()

    _, err = database.Exec(statement)

    if err != nil {
        panic(err)
    }
}

func (db *Database) runQuery(statement string) *sql.Rows {
    database, err := sql.Open("sqlite3", db.File)

    if err != nil {
        panic(err)
    }

    defer database.Close()

    rows, err := database.Query(statement)

    if err != nil {
        panic(err)
    }

    //defer rows.Close() -> should be called by the function consuming the query results, otherwise
    //the values are wiped

    return rows
}

// SetAlias creates an alias for an specific word
func (db *Database) SetAlias(word string, alias string, channel string) (aliasCreated int) {
    aliasConfigured := db.aliasConfigured(word, channel)
    if !aliasConfigured {
        aliasExists := db.GetAlias(word, channel)
        if len(aliasExists) <= 0 {
            //Alias does not exist, run insert
            aliasInit := "INSERT INTO alias(alias, word, channel) values (\"" + alias + "\",\"" + word + "\",\"" + channel + "\")"
            db.runStatement(aliasInit)
            aliasCreated = 0
        } else {
            log.Printf("Word %s already has alias %s configured on channel %s", word, alias, channel)
            aliasCreated = 1
        }
    } else {
        aliasCreated = 2
    }
    
    return aliasCreated
}

func (db *Database) aliasConfigured(alias string, channel string) (aliasConfigured bool) {
    query := "SELECT alias from alias where alias == '" + alias + "' AND channel == '" + channel + "';"
    rows := db.runQuery(query)
    // we have to close the rows
    defer rows.Close()
    aliasConfigured = true
    var result string
    for rows.Next() {
        err := rows.Scan(&result)
        if err != nil {
            panic(err)
        }
    }
    if len(result) <= 0 {
        aliasConfigured = false
    }

    return aliasConfigured
}

// GetAlias returns a defined alias for an specific word
func (db *Database) GetAlias(word string, channel string) string {

    query := "SELECT alias FROM alias WHERE word == '" + word + "' AND channel == '" + channel + "';"
    rows := db.runQuery(query)
    // we have to close the rows
    defer rows.Close()

    var result string
    for rows.Next() {
        err := rows.Scan(&result)
        if err != nil {
            panic(err)
        }
    }

    return result
}

// GetAdmins returns admins for a given channel
func (db *Database) GetAdmins(channel string) []string {
    query := "SELECT user FROM admins WHERE channel == '" + channel + "';"
    rows := db.runQuery(query)
    // we have to close the rows
    defer rows.Close()

    var result string
    var admins []string

    for rows.Next() {
        err := rows.Scan(&result)
        if err != nil {
            panic(err)
        }
        admins = append(admins, result)
    }

    return admins
}

// CreateAdmin Creates a new admin in the database
func (db *Database) CreateAdmin(channel string, user string) {
    adminInsert := "INSERT INTO admins(channel, user) values (\"" + channel + "\",\"" + user + "\")"
    db.runStatement(adminInsert)
}

// DeleteAdmin Deletes an admin from the database
func (db *Database) DeleteAdmin(channel string, user string) {
    adminDelete := "DELETE FROM admins WHERE user == '" + user + "' AND channel == '" + channel + "';"
    db.runStatement(adminDelete)
}

// GetSetting returns the value for a given setting
func (db *Database) GetSetting(channel string, setting string) string {
    query := "SELECT value FROM settings WHERE setting == '" + setting + "' AND channel == '" + channel + "';"
    rows := db.runQuery(query)
    defer rows.Close()

    var settingValue string
    for rows.Next() {
        err := rows.Scan(&settingValue)
        if err != nil {
            panic(err)
        }
    }
    return settingValue
}

// SetSetting creates or updates a setting in a given channel
func (db *Database) SetSetting(channel string, settingName string, settingValue string) {
    settingExists := db.GetSetting(channel, settingName)

    if len(settingExists) <= 0 {
        //Setting does not exist, run insert
        settingInit := "INSERT INTO settings(channel, setting, value) values (\"" + channel + "\",\"" + settingName + "\"," + settingValue + ")"
        db.runStatement(settingInit)
    } else {
        //Setting does exist, run update
        settingUpdate := "UPDATE settings SET value = " + settingValue + " WHERE setting == '" + settingName + "' AND channel == '" + channel + "';"
        db.runStatement(settingUpdate)
    }
}

// UpdateKarma updates the karma for a given word in a given channel
func (db *Database) UpdateKarma(channel string, word string, karmaCounter int) (finalKarma string, notifyKarma bool) {
    // get current karma
    currentKarma := db.GetCurrentKarma(channel, word)

    log.Printf("Current karma for word %s in channel %s is %d", word, channel, currentKarma)
    // Initialize karma if needed
    if currentKarma == -256256 {
        karmaInit := "INSERT INTO karma(channel, word, karma) values (\"" + channel + "\",\"" + word + "\"," + "0" + ")"
        db.runStatement(karmaInit)
        currentKarma = 0
    }
    // Update karma -> + (+int) = + || + (-int) = -
    currentKarma += karmaCounter
    // Check if we have to notify karma change based on setting
    notifyKarmaSetting := db.GetSetting(channel, "notify_karma")
    
    if len(notifyKarmaSetting) <= 0 {
        notifyKarmaSetting = "1"
    }
    notifyKarmaSettingInt, _ := strconv.Atoi(notifyKarmaSetting)
    
    if currentKarma%notifyKarmaSettingInt == 0 {
        notifyKarma = true
    }
    finalKarma = strconv.Itoa(currentKarma)
    karmaUpdate := "UPDATE karma SET karma = " + finalKarma + " WHERE word == '" + word + "' AND channel == '" + channel + "';"
    db.runStatement(karmaUpdate)
    log.Printf("Karma for word %s in channel %s updated to %s", word, channel, finalKarma)
    return finalKarma, notifyKarma
}

// GetCurrentKarma returns the current karma for an specific word
func (db *Database) GetCurrentKarma(channel string, word string) int {
    query := "SELECT karma FROM karma WHERE word == '" + word + "' AND channel == '" + channel + "';"
    rows := db.runQuery(query)
    defer rows.Close()

    var result int = -256256
    for rows.Next() {
        err := rows.Scan(&result)
        if err != nil {
            panic(err)
        }
    }
    return result
}