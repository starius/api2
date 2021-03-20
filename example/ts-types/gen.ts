/* eslint-disable */
// prettier-disable
// prettier-ignore
// Code generated by api2. DO NOT EDIT.
import {route} from "./utils"

export const api = {
example: {
	IEchoService: {
			Hello: route<example.HelloRequest, example.HelloResponse>(
				"POST", "/hello",
				{"query":["key"]},
				{"header":["session"]}),
			Echo: route<example.EchoRequest, example.EchoResponse>(
				"POST", "/echo",
				{"header":["session"],"json":["text","dir","items","maps"]},
				{"json":["text"]}),
	},
},
}
export const DirectionEnum  = {
    "East": 1,
    "North": 0,
    "South": 2,
    "West": 3,
} as const

export declare namespace example {

export type HelloRequest = {
	key?: string
}


export type HelloResponse = {
	session?: string
}


export type EchoRequest = {
	session?: string
	text: string
	dir: example.Direction
	items: Array<example.CustomType2> | null
	maps: Record<string, example.Direction> |  null
}


export type Direction = typeof DirectionEnum[keyof typeof DirectionEnum]

export type CustomType2 =  example.UserSettings & {
}


export type UserSettings = Record<string, any> |  null
// EchoResponse.
export type EchoResponse = {
	text: string // field comment.
}


export type CustomType =  example.UserSettings & {
}

}
