package globals

//Registration
const KEYGEN = 1       //Generating Cryptographic Keys
const PRECAN_REG = 2   //Doing a Precanned Registration (Not Secure)
const UID_GEN = 3      //Generating User ID
const PERM_REG = 4     //Validating User Identity With Permissioning Server
const NODE_REG = 5     //Registering with Nodes
const REG_FAIL = 6     //Failed to Register with Nodes
const SECURE_STORE = 7 //Creating Local Secure Session
const SAVE = 8         //Storing Session
const REG_COMPLETE = 9 //Registration Complete
//UDB registration
const UDB_KEY = 10 //Pushing Cryptographic Material to the User Discovery Bot
const UDB_REG = 11 //Registering User with the User Discovery Bot
//UDB Search
const UDB_SEARCH_LOOK = 12        //Searching for User in User Discovery
const UDB_SEARCH_GETKEY = 13      //Getting Keying Material From User Discovery
const UDB_SEARCH_BUILD_CREDS = 14 //Building secure end to end relationship
