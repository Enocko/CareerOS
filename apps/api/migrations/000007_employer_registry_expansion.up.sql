-- Employer registry expansion v1: 29 verified boards from source coverage audit

INSERT INTO opportunity_sources (id, name, source_type, adapter, config, enabled, sync_interval_minutes)
VALUES
    (
        'c3000000-0000-4000-8000-000000000040',
        'Greenhouse · Affirm',
        'api',
        'greenhouse',
        '{"board_token":"affirm","employer_name":"Affirm","source_url":"https://boards.greenhouse.io/affirm","tags":["fintech"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000041',
        'Greenhouse · Airbnb',
        'api',
        'greenhouse',
        '{"board_token":"airbnb","employer_name":"Airbnb","source_url":"https://boards.greenhouse.io/airbnb","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000042',
        'Greenhouse · Amplitude',
        'api',
        'greenhouse',
        '{"board_token":"amplitude","employer_name":"Amplitude","source_url":"https://boards.greenhouse.io/amplitude","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000043',
        'Greenhouse · Anthropic',
        'api',
        'greenhouse',
        '{"board_token":"anthropic","employer_name":"Anthropic","source_url":"https://boards.greenhouse.io/anthropic","tags":["technology","ai"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000044',
        'Greenhouse · Asana',
        'api',
        'greenhouse',
        '{"board_token":"asana","employer_name":"Asana","source_url":"https://boards.greenhouse.io/asana","tags":["technology","product"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000045',
        'Greenhouse · Brex',
        'api',
        'greenhouse',
        '{"board_token":"brex","employer_name":"Brex","source_url":"https://boards.greenhouse.io/brex","tags":["fintech"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000046',
        'Greenhouse · Chime',
        'api',
        'greenhouse',
        '{"board_token":"chime","employer_name":"Chime","source_url":"https://boards.greenhouse.io/chime","tags":["fintech"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000047',
        'Greenhouse · Databricks',
        'api',
        'greenhouse',
        '{"board_token":"databricks","employer_name":"Databricks","source_url":"https://boards.greenhouse.io/databricks","tags":["technology","data"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000048',
        'Greenhouse · Elastic',
        'api',
        'greenhouse',
        '{"board_token":"elastic","employer_name":"Elastic","source_url":"https://boards.greenhouse.io/elastic","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000049',
        'Greenhouse · GitLab',
        'api',
        'greenhouse',
        '{"board_token":"gitlab","employer_name":"GitLab","source_url":"https://boards.greenhouse.io/gitlab","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000050',
        'Greenhouse · Instacart',
        'api',
        'greenhouse',
        '{"board_token":"instacart","employer_name":"Instacart","source_url":"https://boards.greenhouse.io/instacart","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000051',
        'Greenhouse · Intercom',
        'api',
        'greenhouse',
        '{"board_token":"intercom","employer_name":"Intercom","source_url":"https://boards.greenhouse.io/intercom","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000052',
        'Greenhouse · MongoDB',
        'api',
        'greenhouse',
        '{"board_token":"mongodb","employer_name":"MongoDB","source_url":"https://boards.greenhouse.io/mongodb","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000053',
        'Greenhouse · Okta',
        'api',
        'greenhouse',
        '{"board_token":"okta","employer_name":"Okta","source_url":"https://boards.greenhouse.io/okta","tags":["technology","cybersecurity"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000054',
        'Greenhouse · Postman',
        'api',
        'greenhouse',
        '{"board_token":"postman","employer_name":"Postman","source_url":"https://boards.greenhouse.io/postman","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000055',
        'Greenhouse · Reddit',
        'api',
        'greenhouse',
        '{"board_token":"reddit","employer_name":"Reddit","source_url":"https://boards.greenhouse.io/reddit","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000056',
        'Greenhouse · Robinhood',
        'api',
        'greenhouse',
        '{"board_token":"robinhood","employer_name":"Robinhood","source_url":"https://boards.greenhouse.io/robinhood","tags":["fintech"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000057',
        'Greenhouse · Scale AI',
        'api',
        'greenhouse',
        '{"board_token":"scaleai","employer_name":"Scale AI","source_url":"https://boards.greenhouse.io/scaleai","tags":["technology","ai"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000058',
        'Greenhouse · Twilio',
        'api',
        'greenhouse',
        '{"board_token":"twilio","employer_name":"Twilio","source_url":"https://boards.greenhouse.io/twilio","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000059',
        'Greenhouse · Vercel',
        'api',
        'greenhouse',
        '{"board_token":"vercel","employer_name":"Vercel","source_url":"https://boards.greenhouse.io/vercel","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000060',
        'Greenhouse · Zscaler',
        'api',
        'greenhouse',
        '{"board_token":"zscaler","employer_name":"Zscaler","source_url":"https://boards.greenhouse.io/zscaler","tags":["cybersecurity"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000061',
        'Greenhouse · DoorDash',
        'api',
        'greenhouse',
        '{"board_token":"doordashusa","employer_name":"DoorDash","source_url":"https://boards.greenhouse.io/doordashusa","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000062',
        'Greenhouse · SentinelOne',
        'api',
        'greenhouse',
        '{"board_token":"sentinellabs","employer_name":"SentinelOne","source_url":"https://boards.greenhouse.io/sentinellabs","tags":["cybersecurity"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000070',
        'Ashby · Benchling',
        'api',
        'ashby',
        '{"board_token":"benchling","employer_name":"Benchling","source_url":"https://jobs.ashbyhq.com/benchling","tags":["healthtech"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000071',
        'Ashby · Cohere',
        'api',
        'ashby',
        '{"board_token":"cohere","employer_name":"Cohere","source_url":"https://jobs.ashbyhq.com/cohere","tags":["technology","ai"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000072',
        'Ashby · Confluent',
        'api',
        'ashby',
        '{"board_token":"confluent","employer_name":"Confluent","source_url":"https://jobs.ashbyhq.com/confluent","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000073',
        'Ashby · Sentry',
        'api',
        'ashby',
        '{"board_token":"sentry","employer_name":"Sentry","source_url":"https://jobs.ashbyhq.com/sentry","tags":["technology"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000074',
        'Ashby · Snowflake',
        'api',
        'ashby',
        '{"board_token":"snowflake","employer_name":"Snowflake","source_url":"https://jobs.ashbyhq.com/snowflake","tags":["technology","data"]}'::jsonb,
        true,
        360
    ),
    (
        'c3000000-0000-4000-8000-000000000080',
        'Lever · Veeva Systems',
        'api',
        'lever',
        '{"board_token":"veeva","employer_name":"Veeva Systems","source_url":"https://jobs.lever.co/veeva","tags":["healthtech"]}'::jsonb,
        true,
        360
    );

INSERT INTO employer_boards (id, employer_name, ats_provider, board_token, source_url, tags, enabled, opportunity_source_id)
VALUES
    ('c4000000-0000-4000-8000-000000000040', 'Affirm',      'greenhouse', 'affirm',        'https://www.affirm.com/careers',                              ARRAY['fintech'],                 true, 'c3000000-0000-4000-8000-000000000040'),
    ('c4000000-0000-4000-8000-000000000041', 'Airbnb',      'greenhouse', 'airbnb',        'https://careers.airbnb.com/',                                 ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000041'),
    ('c4000000-0000-4000-8000-000000000042', 'Amplitude',   'greenhouse', 'amplitude',     'https://amplitude.com/careers',                               ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000042'),
    ('c4000000-0000-4000-8000-000000000043', 'Anthropic',   'greenhouse', 'anthropic',     'https://www.anthropic.com/careers',                             ARRAY['technology','ai'],         true, 'c3000000-0000-4000-8000-000000000043'),
    ('c4000000-0000-4000-8000-000000000044', 'Asana',       'greenhouse', 'asana',         'https://asana.com/jobs',                                        ARRAY['technology','product'],    true, 'c3000000-0000-4000-8000-000000000044'),
    ('c4000000-0000-4000-8000-000000000045', 'Brex',        'greenhouse', 'brex',          'https://www.brex.com/careers',                                  ARRAY['fintech'],                 true, 'c3000000-0000-4000-8000-000000000045'),
    ('c4000000-0000-4000-8000-000000000046', 'Chime',       'greenhouse', 'chime',         'https://www.chime.com/careers/',                                ARRAY['fintech'],                 true, 'c3000000-0000-4000-8000-000000000046'),
    ('c4000000-0000-4000-8000-000000000047', 'Databricks',  'greenhouse', 'databricks',    'https://www.databricks.com/company/careers',                    ARRAY['technology','data'],       true, 'c3000000-0000-4000-8000-000000000047'),
    ('c4000000-0000-4000-8000-000000000048', 'Elastic',     'greenhouse', 'elastic',       'https://www.elastic.co/careers',                                ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000048'),
    ('c4000000-0000-4000-8000-000000000049', 'GitLab',      'greenhouse', 'gitlab',        'https://about.gitlab.com/jobs/',                                ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000049'),
    ('c4000000-0000-4000-8000-000000000050', 'Instacart',   'greenhouse', 'instacart',     'https://instacart.careers/',                                    ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000050'),
    ('c4000000-0000-4000-8000-000000000051', 'Intercom',    'greenhouse', 'intercom',      'https://www.intercom.com/careers',                              ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000051'),
    ('c4000000-0000-4000-8000-000000000052', 'MongoDB',     'greenhouse', 'mongodb',       'https://www.mongodb.com/careers',                               ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000052'),
    ('c4000000-0000-4000-8000-000000000053', 'Okta',        'greenhouse', 'okta',          'https://www.okta.com/company/careers/',                         ARRAY['technology','cybersecurity'], true, 'c3000000-0000-4000-8000-000000000053'),
    ('c4000000-0000-4000-8000-000000000054', 'Postman',     'greenhouse', 'postman',       'https://www.postman.com/company/careers/',                      ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000054'),
    ('c4000000-0000-4000-8000-000000000055', 'Reddit',      'greenhouse', 'reddit',        'https://www.redditinc.com/careers',                             ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000055'),
    ('c4000000-0000-4000-8000-000000000056', 'Robinhood',   'greenhouse', 'robinhood',     'https://careers.robinhood.com/',                                ARRAY['fintech'],                 true, 'c3000000-0000-4000-8000-000000000056'),
    ('c4000000-0000-4000-8000-000000000057', 'Scale AI',    'greenhouse', 'scaleai',       'https://scale.com/careers',                                     ARRAY['technology','ai'],         true, 'c3000000-0000-4000-8000-000000000057'),
    ('c4000000-0000-4000-8000-000000000058', 'Twilio',      'greenhouse', 'twilio',        'https://www.twilio.com/company/jobs',                           ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000058'),
    ('c4000000-0000-4000-8000-000000000059', 'Vercel',      'greenhouse', 'vercel',        'https://vercel.com/careers',                                    ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000059'),
    ('c4000000-0000-4000-8000-000000000060', 'Zscaler',     'greenhouse', 'zscaler',       'https://www.zscaler.com/careers',                               ARRAY['cybersecurity'],           true, 'c3000000-0000-4000-8000-000000000060'),
    ('c4000000-0000-4000-8000-000000000061', 'DoorDash',    'greenhouse', 'doordashusa',   'https://careersatdoordash.com/',                                ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000061'),
    ('c4000000-0000-4000-8000-000000000062', 'SentinelOne', 'greenhouse', 'sentinellabs',  'https://www.sentinelone.com/careers/',                          ARRAY['cybersecurity'],           true, 'c3000000-0000-4000-8000-000000000062'),
    ('c4000000-0000-4000-8000-000000000070', 'Benchling',   'ashby',      'benchling',     'https://www.benchling.com/careers/',                            ARRAY['healthtech'],              true, 'c3000000-0000-4000-8000-000000000070'),
    ('c4000000-0000-4000-8000-000000000071', 'Cohere',      'ashby',      'cohere',        'https://cohere.com/careers',                                    ARRAY['technology','ai'],         true, 'c3000000-0000-4000-8000-000000000071'),
    ('c4000000-0000-4000-8000-000000000072', 'Confluent',   'ashby',      'confluent',     'https://careers.confluent.io/',                                 ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000072'),
    ('c4000000-0000-4000-8000-000000000073', 'Sentry',      'ashby',      'sentry',        'https://sentry.io/careers/',                                    ARRAY['technology'],              true, 'c3000000-0000-4000-8000-000000000073'),
    ('c4000000-0000-4000-8000-000000000074', 'Snowflake',   'ashby',      'snowflake',     'https://careers.snowflake.com/',                                ARRAY['technology','data'],       true, 'c3000000-0000-4000-8000-000000000074'),
    ('c4000000-0000-4000-8000-000000000080', 'Veeva Systems','lever',      'veeva',         'https://careers.veeva.com/',                                    ARRAY['healthtech'],              true, 'c3000000-0000-4000-8000-000000000080');
