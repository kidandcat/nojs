module github.com/jairo/nojs-landing

go 1.21

require (
	github.com/jairo/mavis/nojs/demo v0.0.0
	github.com/kidandcat/nojs v0.0.0
	maragu.dev/gomponents v1.1.0
)

replace github.com/kidandcat/nojs => ../

replace github.com/jairo/mavis/nojs/demo => ../demo
