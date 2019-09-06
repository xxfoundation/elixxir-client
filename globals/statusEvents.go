package globals

//Registration
const REG_KEYGEN = 1       //Generating Cryptographic Keys
const REG_PRECAN = 2       //Doing a Precanned Registration (Not Secure)
const REG_UID_GEN = 3      //Generating User ID
const REG_PERM = 4         //Validating User Identity With Permissioning Server
const REG_NODE = 5         //Registering with Nodes
const REG_FAIL = 6         //Failed to Register with Nodes
const REG_SECURE_STORE = 7 //Creating Local Secure Session
const REG_SAVE = 8         //Storing Session
//UDB registration
const UDB_REG_PUSHKEY = 9   //Pushing Cryptographic Material to the User Discovery Bot
const UDB_REG_PUSHUSER = 10 //Registering User with the User Discovery Bot
//UDB Search
const UDB_SEARCH_LOOK = 11        //Searching for User in User Discovery
const UDB_SEARCH_GETKEY = 12      //Getting Keying Material From User Discovery
const UDB_SEARCH_BUILD_CREDS = 13 //Building secure end to end relationship
