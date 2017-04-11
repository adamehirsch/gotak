## GOTAK

More of a "Baby's First Golang" project, really, than anything serious. An attempt to write an implementation of [Tak](http://cheapass.com/tak/), a game designed by James Garfield and Patrick Rothfuss.

Rules can [be found here](http://cheapass.com/wp-content/uploads/2017/01/TakShortRules.pdf)

### TODO

- ~~Coordinates for boards~~
  - ~~a, b, c, d on X-axis~~
  - ~~1, 2, 3, 4 ascending on Y-axis~~
- ~~Concept of a player~~
  - has a color (black or white)
  - has some kind of authentication, someday
- Player actions
  - ~~PLACE~~
    - ~~needed info:~~
      - color
      - (color should be validated against the player's color)
      - ~~empty square coordinates~~
      - ~~orientation ("flat", "wall", "capstone")~~
    - validate:
      - ~~destination square is empty~~
      - ~~player has not played more than their allowed number of~~ pieces
      - ... or capstones
  - MOVE
    - First moves of the game involve placing one stone of the opposite color. Bah.
    - needed info:
      - ~~Origin square~~
      - ~~direction of move (+, -, >, <)~~
      - ~~Number of pieces from stack~~
      - ~~array of number of pieces to drop off at each square ( [1,2,1,...])~~
    - validate:
      - requested number of pieces to move does not exceed the size of the origin stack
      - requested number of pieces does not exceed the board's "carry limit", which is the board size
      - drop sequence must drop at least one piece off at every space
      - move may not leave the board
      - move may not cross a standing stone
        - UNLESS it's a capstone, by itself, landing on a standing stone in which case the standing stone changes its orientation to "flat"
- Game end detection
  - player has run out of stones
  - board has no blank spaces left
  - board has a road
  - board has a road for each player (unusual end condition, win goes to active player)
- Backup file storage
  - on a successful move, atomically save the board state (and history?) to disk
- authentication
  - cookies
- Hoo, boy: maybe an actual Parser for *Portable Tak Notation* (PTN) https://www.reddit.com/r/Tak/wiki/portable_tak_notation
