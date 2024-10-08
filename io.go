package proc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
)

const errPrefix = "!!<ERR" + "OR>!!"

func errOr(err, err2 error) error {
	if err == nil {
		err = err2
	}
	return err
}

func Get(url string) *Proc {
	return Fun(func(w io.Writer) error {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, resp.Body)
		return errOr(err, resp.Body.Close())
	})
}

func Lis(port int, f func(*Proc) *Proc) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		f(Err(err))
		return
	}
	for {
		conn, err := lis.Accept()
		if err != nil {
			f(Err(err))
			return
		}
		go func() {
			p := f(Fun(func(w io.Writer) error {
				_, err := io.Copy(w, conn)
				return err
			}))
			if _, err := io.Copy(conn, p); err != nil {
				fmt.Fprint(conn, errPrefix, errors.Unwrap(err))
			}
			conn.Close()
		}()
	}
}

func Open(name string) *Proc {
	return Fun(func(w io.Writer) error {
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, f)
		return errOr(err, f.Close())
	})
}

func (p *Proc) Post(url, contentType string) *Proc {
	return Fun(func(w io.Writer) error {
		resp, err := http.Post(url, contentType, p)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, resp.Body)
		return errOr(err, resp.Body.Close())
	})
}

func (p *Proc) Put(name string) *Proc {
	return Fun(func(io.Writer) error {
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, p)
		return errOr(err, f.Close())
	})
}

func (p *Proc) Run(stdout, stderr io.Writer) {
	if _, err := io.Copy(stdout, p); err != nil {
		fmt.Fprintln(stderr, err)
	}
}

func (p *Proc) Send(addr string) *Proc {
	return Fun(func(w io.Writer) error {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return err
		}
		ch1 := make(chan error)
		ch2 := make(chan error)
		go func() {
			var buf []byte
			for r, m := bufio.NewReader(conn), 0; ; {
				switch c, err := r.ReadByte(); {
				case err != nil:
					if m < len(errPrefix) {
						w.Write([]byte(errPrefix[:m]))
					}
					if err == io.EOF {
						if err = nil; m == len(errPrefix) {
							err = errors.New(string(buf))
						}
					}
					ch1 <- err
					return
				case m == len(errPrefix):
					buf = append(buf, c)
				case c == errPrefix[m]:
					m++
				default:
					w.Write(append([]byte(errPrefix[:m]), c))
					m = 0
				}
			}
		}()
		go func() {
			_, err := io.Copy(conn, p)
			ch2 <- errOr(err, conn.(*net.TCPConn).CloseWrite())
		}()
		select {
		case err = <-ch1:
		case err = <-ch2:
			if err == nil {
				err = <-ch1
			}
		}
		return errOr(err, conn.Close())
	})
}

func (p *Proc) Std() {
	p.Run(os.Stdout, os.Stderr)
}
