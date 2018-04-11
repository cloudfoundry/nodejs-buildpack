# figlet-cli

> A command line interface for the figlet.js library.

```
  _______  __     _______   ___       _______  ___________  
 /"     "||" \   /" _   "| |"  |     /"     "|("     _   ") 
(: ______)||  | (: ( \___) ||  |    (: ______) )__/  \\__/  
 \/    |  |:  |  \/ \      |:  |     \/    |      \\_ /     
 // ___)  |.  |  //  \ ___  \  |___  // ___)_     |.  |     
(:  (     /\  |\(:   _(  _|( \_|:  \(:      "|    \:  |     
 \__/    (__\_|_)\_______)  \_______)\_______)     \__|     
                                                            
 ```

## Getting Started

Install this globally and you'll have access to the figlet.js library on the command line:

```shell
npm install -g figlet-cli
```

## Usage Examples

#### Default Options
Below is a simple example that uses the default options.

```shell
figlet "hello world"
```

#### Custom Options
Below is an example that uses a custom font.

```shell
figlet -f "Dancing Font" "Hi"
```

## Goals

Eventually I think it would be nice for this app to have to same command line interface as the C-based app.

## Credits

This was originally submitted to [figlet.js](https://github.com/patorjk/figlet.js) by [timhudson](https://github.com/timhudson). It's been broken out as a spearate project so users can control which figlet they want to use on the command line (i.e., so installing the figlet.js library globally wont conflict with the C-based command line figlet app).

## Release History
* 2014.01.02 v0.1.0 Initial release.
