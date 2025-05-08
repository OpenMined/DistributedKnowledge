declare module 'unzipper' {
  export function Extract(options: { path: string }): NodeJS.WritableStream
  export namespace Open {
    export function file(path: string): Promise<Directory>
  }
  export interface Directory {
    files: Array<{ path: string; type: string; buffer: () => Promise<Buffer> }>
    extract: (options: { path: string }) => Promise<void>
  }
}
