-- F1 API database schema

CREATE TABLE IF NOT EXISTS circuits (
    circuit_id  SERIAL PRIMARY KEY,
    ref         VARCHAR(100) UNIQUE NOT NULL,
    name        VARCHAR(255) NOT NULL,
    location    VARCHAR(255),
    country     VARCHAR(100),
    lat         NUMERIC(9, 6),
    lng         NUMERIC(9, 6),
    url         TEXT
);

CREATE TABLE IF NOT EXISTS constructors (
    constructor_id  SERIAL PRIMARY KEY,
    ref             VARCHAR(100) UNIQUE NOT NULL,
    name            VARCHAR(255) NOT NULL,
    nationality     VARCHAR(100),
    url             TEXT
);

CREATE TABLE IF NOT EXISTS drivers (
    driver_id    SERIAL PRIMARY KEY,
    ref          VARCHAR(100) UNIQUE NOT NULL,
    number       INT,
    code         CHAR(3),
    forename     VARCHAR(100) NOT NULL,
    surname      VARCHAR(100) NOT NULL,
    date_of_birth DATE,
    nationality  VARCHAR(100),
    url          TEXT
);

CREATE TABLE IF NOT EXISTS races (
    race_id     SERIAL PRIMARY KEY,
    season      INT NOT NULL,
    round       INT NOT NULL,
    circuit_id  INT NOT NULL REFERENCES circuits(circuit_id),
    name        VARCHAR(255) NOT NULL,
    date        DATE,
    time        TIME,
    url         TEXT,
    UNIQUE (season, round)
);

CREATE TABLE IF NOT EXISTS results (
    result_id           SERIAL PRIMARY KEY,
    race_id             INT NOT NULL REFERENCES races(race_id),
    driver_id           INT NOT NULL REFERENCES drivers(driver_id),
    constructor_id      INT NOT NULL REFERENCES constructors(constructor_id),
    grid                INT NOT NULL DEFAULT 0,
    position            INT,
    position_text       VARCHAR(10) NOT NULL,
    position_order      INT NOT NULL,
    points              NUMERIC(5, 2) NOT NULL DEFAULT 0,
    laps                INT NOT NULL DEFAULT 0,
    time                VARCHAR(20),
    status              VARCHAR(100),
    fastest_lap_rank    INT,
    fastest_lap_time    VARCHAR(20),
    fastest_lap_speed   VARCHAR(20)
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_races_season ON races(season);
CREATE INDEX IF NOT EXISTS idx_results_race ON results(race_id);
CREATE INDEX IF NOT EXISTS idx_results_driver ON results(driver_id);
CREATE INDEX IF NOT EXISTS idx_results_constructor ON results(constructor_id);
