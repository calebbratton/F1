-- Example seed data to get you started
-- Run after 001_init.sql

INSERT INTO seasons (year, url) VALUES
  (2024, 'https://en.wikipedia.org/wiki/2024_Formula_One_World_Championship'),
  (2023, 'https://en.wikipedia.org/wiki/2023_Formula_One_World_Championship')
ON CONFLICT DO NOTHING;

INSERT INTO circuits (ref, name, location, country, lat, lng, url) VALUES
  ('bahrain',   'Bahrain International Circuit',  'Sakhir',      'Bahrain',     26.0325,  50.5106,  'https://en.wikipedia.org/wiki/Bahrain_International_Circuit'),
  ('jeddah',    'Jeddah Corniche Circuit',         'Jeddah',      'Saudi Arabia',21.6319,  39.1044,  'https://en.wikipedia.org/wiki/Jeddah_Street_Circuit'),
  ('albert_park','Albert Park Grand Prix Circuit', 'Melbourne',   'Australia',   -37.8497, 144.968,  'https://en.wikipedia.org/wiki/Albert_Park_Circuit'),
  ('monza',     'Autodromo Nazionale di Monza',    'Monza',       'Italy',       45.6156,   9.2811,  'https://en.wikipedia.org/wiki/Autodromo_Nazionale_Monza'),
  ('silverstone','Silverstone Circuit',            'Silverstone', 'UK',          52.0786,  -1.01694, 'https://en.wikipedia.org/wiki/Silverstone_Circuit')
ON CONFLICT DO NOTHING;

INSERT INTO constructors (ref, name, nationality, url) VALUES
  ('red_bull',  'Red Bull',          'Austrian',  'https://en.wikipedia.org/wiki/Red_Bull_Racing'),
  ('ferrari',   'Ferrari',           'Italian',   'https://en.wikipedia.org/wiki/Scuderia_Ferrari'),
  ('mercedes',  'Mercedes',          'German',    'https://en.wikipedia.org/wiki/Mercedes-AMG_Petronas_F1_Team'),
  ('mclaren',   'McLaren',           'British',   'https://en.wikipedia.org/wiki/McLaren'),
  ('aston_martin','Aston Martin',    'British',   'https://en.wikipedia.org/wiki/Aston_Martin_in_Formula_One')
ON CONFLICT DO NOTHING;

INSERT INTO drivers (ref, number, code, forename, surname, date_of_birth, nationality, url) VALUES
  ('max_verstappen',  1,   'VER', 'Max',      'Verstappen',  '1997-09-30', 'Dutch',      'https://en.wikipedia.org/wiki/Max_Verstappen'),
  ('leclerc',        16,   'LEC', 'Charles',  'Leclerc',     '1997-10-16', 'Monegasque', 'https://en.wikipedia.org/wiki/Charles_Leclerc'),
  ('hamilton',       44,   'HAM', 'Lewis',    'Hamilton',    '1985-01-07', 'British',    'https://en.wikipedia.org/wiki/Lewis_Hamilton'),
  ('norris',          4,   'NOR', 'Lando',    'Norris',      '1999-11-13', 'British',    'https://en.wikipedia.org/wiki/Lando_Norris'),
  ('alonso',         14,   'ALO', 'Fernando', 'Alonso',      '1981-07-29', 'Spanish',    'https://en.wikipedia.org/wiki/Fernando_Alonso'),
  ('perez',          11,   'PER', 'Sergio',   'Pérez',       '1990-01-26', 'Mexican',    'https://en.wikipedia.org/wiki/Sergio_P%C3%A9rez')
ON CONFLICT DO NOTHING;
