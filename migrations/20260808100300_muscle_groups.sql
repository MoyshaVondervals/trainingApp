-- +goose Up
CREATE TABLE muscle_groups (
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(50)  NOT NULL UNIQUE,
    name       VARCHAR(100) NOT NULL,
    parent_id  BIGINT       NOT NULL REFERENCES muscle_regions(id) ON DELETE RESTRICT,
    sort_order INT          NOT NULL DEFAULT 0,
    UNIQUE (parent_id, name)
);

CREATE INDEX muscle_groups_parent_idx ON muscle_groups (parent_id);

INSERT INTO muscle_groups (code, name, parent_id, sort_order)
SELECT v.code, v.name, r.id, v.sort_order
FROM (VALUES
    ('chest_upper',          'Верх груди (ключичная часть)',              'chest',     11),
    ('chest_middle',         'Середина груди (грудинно-рёберная часть)',  'chest',     12),
    ('chest_lower',          'Низ груди (брюшная часть)',                 'chest',     13),
    ('pectoralis_minor',     'Малая грудная',                             'chest',     14),
    ('serratus_anterior',    'Передняя зубчатая',                         'chest',     15),

    ('lats',                 'Широчайшие',                                'back',      21),
    ('traps_upper',          'Трапеция, верх',                            'back',      22),
    ('traps_middle',         'Трапеция, середина',                        'back',      23),
    ('traps_lower',          'Трапеция, низ',                             'back',      24),
    ('rhomboids',            'Ромбовидные',                               'back',      25),
    ('teres_major',          'Большая круглая',                           'back',      26),
    ('erector_spinae',       'Разгибатели спины',                         'back',      27),

    ('delts_front',          'Передняя дельта',                           'shoulders', 31),
    ('delts_side',           'Средняя дельта',                            'shoulders', 32),
    ('delts_rear',           'Задняя дельта',                             'shoulders', 33),
    ('supraspinatus',        'Надостная',                                 'shoulders', 34),
    ('infraspinatus',        'Подостная',                                 'shoulders', 35),
    ('teres_minor',          'Малая круглая',                             'shoulders', 36),
    ('subscapularis',        'Подлопаточная',                             'shoulders', 37),

    ('biceps',               'Бицепс',                                    'arms',      41),
    ('brachialis',           'Брахиалис',                                 'arms',      42),
    ('brachioradialis',      'Плечелучевая',                              'arms',      43),
    ('triceps_long',         'Трицепс, длинная головка',                  'arms',      44),
    ('triceps_lateral',      'Трицепс, латеральная головка',              'arms',      45),
    ('triceps_medial',       'Трицепс, медиальная головка',               'arms',      46),
    ('forearm_flexors',      'Сгибатели предплечья',                      'arms',      47),
    ('forearm_extensors',    'Разгибатели предплечья',                    'arms',      48),

    ('rectus_abdominis',     'Прямая мышца живота',                       'core',      51),
    ('obliques_external',    'Наружные косые',                            'core',      52),
    ('obliques_internal',    'Внутренние косые',                          'core',      53),
    ('transverse_abdominis', 'Поперечная мышца живота',                   'core',      54),
    ('quadratus_lumborum',   'Квадратная мышца поясницы',                 'core',      55),

    ('rectus_femoris',       'Прямая мышца бедра',                        'legs',      61),
    ('vastus_lateralis',     'Латеральная широкая',                       'legs',      62),
    ('vastus_medialis',      'Медиальная широкая',                        'legs',      63),
    ('vastus_intermedius',   'Промежуточная широкая',                     'legs',      64),
    ('biceps_femoris',       'Двуглавая мышца бедра',                     'legs',      65),
    ('semitendinosus',       'Полусухожильная',                           'legs',      66),
    ('semimembranosus',      'Полуперепончатая',                          'legs',      67),
    ('gluteus_maximus',      'Большая ягодичная',                         'legs',      68),
    ('gluteus_medius',       'Средняя ягодичная',                         'legs',      69),
    ('gluteus_minimus',      'Малая ягодичная',                           'legs',      70),
    ('adductors',            'Приводящие',                                'legs',      71),
    ('tensor_fasciae_latae', 'Напрягатель широкой фасции',                'legs',      72),
    ('gastrocnemius',        'Икроножная',                                'legs',      73),
    ('soleus',               'Камбаловидная',                             'legs',      74),
    ('tibialis_anterior',    'Передняя большеберцовая',                   'legs',      75),

    ('sternocleidomastoid',  'Грудино-ключично-сосцевидная',              'neck',      81),
    ('neck_extensors',       'Разгибатели шеи',                           'neck',      82)
) AS v(code, name, region_code, sort_order)
JOIN muscle_regions r ON r.code = v.region_code
ON CONFLICT (code) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS muscle_groups CASCADE;
