# go-customicon


# Extended Attributes

### com.apple.FinderInfo

Size: 32 Bytes

SetFile Flags:
    
    Bits
    1  Located on the desktop (allowed on folders)
    2  
    4  Extension is hidden
    8  
    16  
    32  
    64  Shared
    128  No Init
    256  Inited
    512  
    1024  Custom Icon
    2048  "Stationery Pad" file
    4096  System file (name locked)
    8192  Has bundle
    16384 Invisible
    32768 Alias File
    
-

    Byte => Bits   SetFile  Description 
       8 => 1      I | i    Inited - Finder is aware of this file and has given it a location in a window. (allowed on folders)
       8 => 4      C | c    Custom icon (allowed on folders)
       8 => 8      T | t    "Stationery Pad" file
       8 => 16     S | s    System file (name locked)
       8 => 32     B | b    Has bundle
       8 => 64     V | v    Invisible (allowed on folders)
       8 => 128    A | a    Alias file
       
       9 => 1      D | d    Located on the desktop (allowed on folders)
       9 => 16     E | e    Extension is hidden (allowed on folders)
       9 => 64     M | m    Shared (can run multiple times)
       9 => 128    N | n    File has no INIT resource
       
       25 => 128   Z | z    Busy (allowed on folders)


1024+16384

17408
