# htmlc

**Supercharge your HTML with JSX-style components with Go Templates**

## Why ? 
HTML does not have any support for Components (you can do it with [WebComponents](https://developer.mozilla.org/en-US/docs/Web/API/Web_components), but that's another story), which means one has to keep copy pasting same elements everywhere.

`htmlc` allows one to use go templates as `Components` in your html pages

### **Without htmlc**
```html
<button class="bg-blue-200 text-blue-900 px-4 py-2 tracking-wide">Button1</button> <br/>
<button class="bg-blue-200 text-blue-900 px-4 py-2 tracking-wide">Button2</button> <br/>
<button class="bg-blue-200 text-blue-900 px-4 py-2 tracking-wide">Button3</button> <br/>
```

### **With htmlc**
```html
<MyButton>Button1</button> <br/>
<MyButton>Button2</button> <br/>
<MyButton>Button3</button> <br/>
```

> [!NOTE]
> here, **MyButton** is a component (HTML snippet), written with [Go Templates](https://pkg.go.dev/html/template)
> ```html
> {{- define "MyButton" }}
> <button class="bg-blue-200 text-blue-900 px-4 py-2 tracking-wide">
>  <Children />
> </button>
> {{- end }}
> ```

## Getting Started 🚀

### Installation

| OS           | Command                                           |
|:---:         |:---:                                              |
| Go           | `go install github.com/nxtcoder17/htmlc@latest`   | 

### Setup

```sh
htmlc init
```

> [!NOTE]
> It creates
> | file                                   | description                                                 |
> | :--:                                   | :--:                                                        |
> | [htmlc.yml](./examples/htmlc.yml)      | htmlc default configuration file                            |
> | [components](./examples/components/)   | directory for your components, with some example components |
> | [pages](./examples/pages)              | directory for your html pages and a sample html page        |

### Usage

- You can create your pages in `pages` directory, and it can use any defined templates from `components` directory.
- `htmlc init` will give a few examples of both components and pages, which should guide you to create your own components, and
how to use them in your pages

   

## How it works ?
1. you write your HTML pages and components separately, so that any HTML page can use any component _(also, any component can make use of other components)_.

2. htmlc iterates through all the *.html files in the components directory
    - generates a go file for each component ([template](./pkg/parser/template/printer_template.go.tpl)), kind of coverting each Component into a go code
    - after that, it also generates one more [go file](./pkg/parser/template/generated_template.go.tpl) for the components package, that contains some global variables that previously generated files are using.

3. Post that, it iterates through all html files in pages directory, one by one, and
    - traverses the entire tree, and whenever it finds a tag (like `<MyButton>`) which is not a standard HTML tag, it replaces it with the component template body which was generated in the previous step.
 
## Gallery

- Demo GIF
  ![**See it in Action**](https://github.com/user-attachments/assets/6d92b4ed-a9aa-4619-ad3c-38a6b9ceb746)
