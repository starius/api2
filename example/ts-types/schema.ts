// prettier-ignore 
import * as t from './gen'
import type {JTDSchema} from 'libs/validator'

export const schema =  {
ExampleCustomType: {
  "metadata": {
    "allOf": [
      "ExampleUserSettings"
    ]
  }
} as JTDSchema<t.example.CustomType>,
ExampleCustomType2: {
  "metadata": {
    "allOf": [
      "ExampleUserSettings"
    ]
  }
} as JTDSchema<t.example.CustomType2>,
ExampleDirection: {
  "metadata": {
    "enumType": "int",
    "enumValues": [
      1,
      0,
      2,
      3
    ]
  },
  "type": "int32"
} as JTDSchema<t.example.Direction>,
ExampleEchoRequest: {
  "properties": {
    "dir": {
      "ref": "ExampleDirection"
    },
    "items": {
      "elements": {}
    },
    "maps": {
      "values": {
        "ref": "ExampleDirection"
      }
    },
    "session": {
      "type": "string"
    },
    "text": {
      "type": "string"
    }
  }
} as JTDSchema<t.example.EchoRequest>,
ExampleEchoResponse: {
  "properties": {
    "text": {
      "type": "string"
    }
  }
} as JTDSchema<t.example.EchoResponse>,
ExampleHelloRequest: {
  "properties": {
    "key": {
      "type": "string"
    }
  }
} as JTDSchema<t.example.HelloRequest>,
ExampleHelloResponse: {
  "properties": {
    "session": {
      "type": "string"
    }
  }
} as JTDSchema<t.example.HelloResponse>,
ExampleUserSettings: {} as JTDSchema<t.example.UserSettings>,
}
