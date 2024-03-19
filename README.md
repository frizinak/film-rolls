# Film rolls

A way to record the history of film rolls passing through your camera(s).

## Format

### Definitions

Company
```
Company [company-id]
    [name]
```

Film stock
```
Stock [stock-id]
    [format]
    [name]
    [company-id]
    [iso range]
    [#rolls]
```

Camera
```
Camera [camera-id]
    [brand]
    [model]
```

Development lab
```
Lab [lab-id]
    [name]
```

### Log

Film just loaded in camera:
```
[loaded-in-camera-date] [stock-id] [camera-id]
    [notes]
```

Film just removed from camera:
```
[loaded-in-camera-date] [stock-id] [camera-id] -
    [notes]
```

Film delivered to lab:
```
[loaded-in-camera-date] [stock-id] [camera-id] [lab-id] [lab-in-date]
    [notes]
```

Film developed:
```
[loaded-in-camera-date] [stock-id] [camera-id] [lab-id] [lab-in-date] [lab-out-date] [page-in-film-binder]
    [notes]
```

### Example


![screenshot](https://raw.githubusercontent.com/frizinak/film-rolls/dev/.github/term-table.png)


```
Company FUJ
    Fujifilm

Company KOD
    Kodak

Company LOM
    Lomography

Stock C92
    135
    Lomochrome Color '92
    LOM
    400
    20 + 5 + 6

Stock TRQ
    135
    Lomochrome Turquoise
    LOM
    100 400
    3

Stock MTR
    135
    Lomochrome Metropolis
    LOM
    100 400
    8

Stock PUR
    135
    Lomochrome Purple XR
    LOM
    100 400
    1

Stock RSC
    135
    Redscale XR
    LOM
    50-200
    5

Stock LGR
    135
    Lady Grey
    LOM
    400
    20

Stock VTF
    135
    Vision3 250D
    KOD
    250
    8

Stock 200
    135
    200
    FUJ
    200
    8 + 6 + 6

Stock XTR
    135
    Superia X-Tra
    FUJ
    400
    8 + 6 + 6

Camera OM1
    Olympus
    OM-1

Camera OM2
    Olympus
    OM-2n

Camera ZNT
    Зенит
    12СД

Lab MOR
    MORI Film Lab

Lab WTB
    WATANABE - Hanoi - Vietnam

Lab FSL
    De Foto Studio - Leuven

###############################################################################
###############################################################################
###############################################################################

2023-05-19 VTF OM1 WTB 2023-05-21 2023-05-25 0001
    Vietnam

2023-05-21 VTF OM1
    Vietnam

2023-05-23 200 OM1

2023-06-03 XTR OM1 FSL 2023-07-01 2023-07-01 0002

2023-09-27 RSC ZNT

2023-09-27 PUR OM1

2023-10-11 RSC ZNT

2023-10-11 LGR OM1
    Rotterdam

2023-11-14 PUR ZNT
    Rotterdam

2023-10-11 C92 OM2

2023-11-30 C92 OM1
    Rotterdam

2023-12-02 TRQ ZNT

2023-12-10 MTR OM1
```
