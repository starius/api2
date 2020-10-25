/* tslint:disable */
/* eslint-disable */
export namespace types {
  //github.com/zmitry/go2typings/example/types.WeekDay
  export type WeekDay = "mon" | "sun"
  //github.com/zmitry/go2typings/example/types.WeekDay2
  export type WeekDay2 = "3" | "4"
  //github.com/zmitry/go2typings/example/types.T
  export interface T {
    weekday: types.WeekDay;
    weekday2: types.WeekDay2;
    date: string;
  }
  //github.com/zmitry/go2typings/example/types.UserTag
  export interface UserTag {
    tag: string;
  }
  //github.com/zmitry/go2typings/example/types.User
  export interface User {
    firstname: string;
    secondName: string;
    tags: Array<types.UserTag> | null;
  }
}

