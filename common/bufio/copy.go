package bufio

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/buf"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/common/rw"
)

func Copy(destination io.Writer, source io.Reader) (n int64, err error) {
	if source == nil {
		return 0, E.New("nil reader")
	} else if destination == nil {
		return 0, E.New("nil writer")
	}
	originSource := source
	var readCounters, writeCounters []N.CountFunc
	for {
		source, readCounters = N.UnwrapCountReader(source, readCounters)
		destination, writeCounters = N.UnwrapCountWriter(destination, writeCounters)
		if cachedSrc, isCached := source.(N.CachedReader); isCached {
			cachedBuffer := cachedSrc.ReadCached()
			if cachedBuffer != nil {
				if !cachedBuffer.IsEmpty() {
					_, err = destination.Write(cachedBuffer.Bytes())
					if err != nil {
						cachedBuffer.Release()
						return
					}
				}
				cachedBuffer.Release()
				continue
			}
		}

		if srcNetConn, srcIsNetConn := source.(net.Conn); srcIsNetConn {
			srcNetConn.SetDeadline(time.Time{})
		}
		if dstNetConn, dstIsNetConn := destination.(net.Conn); dstIsNetConn {
			dstNetConn.SetDeadline(time.Time{})
		}

		srcSyscallConn, srcIsSyscall := source.(syscall.Conn)
		dstSyscallConn, dstIsSyscall := destination.(syscall.Conn)
		if srcIsSyscall && dstIsSyscall {
			var handled bool
			handled, n, err = copyDirect(srcSyscallConn, dstSyscallConn, readCounters, writeCounters)
			if handled {
				return
			}
		}
		break
	}
	return CopyExtended(originSource, NewExtendedWriter(destination), NewExtendedReader(source), readCounters, writeCounters)
}

func CopyExtended(originSource io.Reader, destination N.ExtendedWriter, source N.ExtendedReader, readCounters []N.CountFunc, writeCounters []N.CountFunc) (n int64, err error) {
	frontHeadroom := N.CalculateFrontHeadroom(destination)
	rearHeadroom := N.CalculateRearHeadroom(destination)
	readWaiter, isReadWaiter := CreateReadWaiter(source)
	if isReadWaiter {
		needCopy := readWaiter.InitializeReadWaiter(N.ReadWaitOptions{
			FrontHeadroom: frontHeadroom,
			RearHeadroom:  rearHeadroom,
			MTU:           N.CalculateMTU(source, destination),
		})
		if !needCopy || common.LowMemory {
			var handled bool
			handled, n, err = copyWaitWithPool(originSource, destination, readWaiter, readCounters, writeCounters)
			if handled {
				return
			}
		}
	}
	return CopyExtendedWithPool(originSource, destination, source, readCounters, writeCounters)
}

func CopyExtendedBuffer(originSource io.Writer, destination N.ExtendedWriter, source N.ExtendedReader, buffer *buf.Buffer, readCounters []N.CountFunc, writeCounters []N.CountFunc) (n int64, err error) {
	buffer.IncRef()
	defer buffer.DecRef()
	frontHeadroom := N.CalculateFrontHeadroom(destination)
	rearHeadroom := N.CalculateRearHeadroom(destination)
	buffer.Resize(frontHeadroom, 0)
	buffer.Reserve(rearHeadroom)
	var notFirstTime bool
	for {
		err = source.ReadBuffer(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
				return
			}
			return
		}
		dataLen := buffer.Len()
		buffer.OverCap(rearHeadroom)
		err = destination.WriteBuffer(buffer)
		if err != nil {
			if !notFirstTime {
				err = N.ReportHandshakeFailure(originSource, err)
			}
			return
		}
		n += int64(dataLen)
		for _, counter := range readCounters {
			counter(int64(dataLen))
		}
		for _, counter := range writeCounters {
			counter(int64(dataLen))
		}
		notFirstTime = true
	}
}

func CopyExtendedWithPool(originSource io.Reader, destination N.ExtendedWriter, source N.ExtendedReader, readCounters []N.CountFunc, writeCounters []N.CountFunc) (n int64, err error) {
	frontHeadroom := N.CalculateFrontHeadroom(destination)
	rearHeadroom := N.CalculateRearHeadroom(destination)
	bufferSize := N.CalculateMTU(source, destination)
	if bufferSize > 0 {
		bufferSize += frontHeadroom + rearHeadroom
	} else {
		bufferSize = buf.BufferSize
	}
	var notFirstTime bool
	for {
		buffer := buf.NewSize(bufferSize)
		buffer.Resize(frontHeadroom, 0)
		buffer.Reserve(rearHeadroom)
		err = source.ReadBuffer(buffer)
		if err != nil {
			buffer.Release()
			if errors.Is(err, io.EOF) {
				err = nil
				return
			}
			return
		}
		dataLen := buffer.Len()
		buffer.OverCap(rearHeadroom)
		err = destination.WriteBuffer(buffer)
		if err != nil {
			buffer.Leak()
			if !notFirstTime {
				err = N.ReportHandshakeFailure(originSource, err)
			}
			return
		}
		n += int64(dataLen)
		for _, counter := range readCounters {
			counter(int64(dataLen))
		}
		for _, counter := range writeCounters {
			counter(int64(dataLen))
		}
		notFirstTime = true
	}
}

func CopyConn(ctx context.Context, source net.Conn, destination net.Conn) error {
	return CopyConnContextList([]context.Context{ctx}, source, destination)
}

func CopyConnContextList(contextList []context.Context, source net.Conn, destination net.Conn) error {
	var err1, err2 error
	wg := sync.WaitGroup{}
	wg.Add(2)
	if _, dstDuplex := common.Cast[rw.WriteCloser](destination); dstDuplex {
		go func() {
			defer wg.Done()
			err1 = common.Error(Copy(destination, source))
			if err1 == nil {
				rw.CloseWrite(destination)
			} else {
				common.Close(destination)
				err1 = E.Cause(err1, "upload")
			}
		}()
	} else {
		go func() {
			defer wg.Done()
			defer common.Close(destination)
			err1 = common.Error(Copy(destination, source))
			if err1 != nil {
				err1 = E.Cause(err1, "upload")
			}
		}()
	}
	if _, srcDuplex := common.Cast[rw.WriteCloser](source); srcDuplex {
		go func() {
			defer wg.Done()
			err2 = common.Error(Copy(source, destination))
			if err2 == nil {
				rw.CloseWrite(source)
			} else {
				common.Close(source)
				err2 = E.Cause(err2, "download")
			}
		}()
	} else {
		go func() {
			defer wg.Done()
			defer common.Close(source)
			err2 = common.Error(Copy(source, destination))
			if err2 != nil {
				err2 = E.Cause(err2, "download")
			}
		}()
	}
	wg.Wait()
	common.Close(source, destination)

	var retErr error
	if err1 != nil {
		retErr = E.Errors(err1)
	}
	if err2 != nil {
		retErr = E.Errors(err2)
	}
	return retErr
}

func CopyPacket(destinationConn N.PacketWriter, source N.PacketReader) (n int64, err error) {
	var readCounters, writeCounters []N.CountFunc
	var cachedPackets []*N.PacketBuffer
	originSource := source
	for {
		source, readCounters = N.UnwrapCountPacketReader(source, readCounters)
		destinationConn, writeCounters = N.UnwrapCountPacketWriter(destinationConn, writeCounters)
		if cachedReader, isCached := source.(N.CachedPacketReader); isCached {
			packet := cachedReader.ReadCachedPacket()
			if packet != nil {
				cachedPackets = append(cachedPackets, packet)
				continue
			}
		}
		break
	}
	if cachedPackets != nil {
		n, err = WritePacketWithPool(originSource, destinationConn, cachedPackets)
		if err != nil {
			return
		}
	}
	frontHeadroom := N.CalculateFrontHeadroom(destinationConn)
	rearHeadroom := N.CalculateRearHeadroom(destinationConn)
	var (
		handled bool
		copeN   int64
	)
	readWaiter, isReadWaiter := CreatePacketReadWaiter(source)
	if isReadWaiter {
		needCopy := readWaiter.InitializeReadWaiter(N.ReadWaitOptions{
			FrontHeadroom: frontHeadroom,
			RearHeadroom:  rearHeadroom,
			MTU:           N.CalculateMTU(source, destinationConn),
		})
		if !needCopy || common.LowMemory {
			handled, copeN, err = copyPacketWaitWithPool(originSource, destinationConn, readWaiter, readCounters, writeCounters, n > 0)
			if handled {
				n += copeN
				return
			}
		}
	}
	copeN, err = CopyPacketWithPool(originSource, destinationConn, source, readCounters, writeCounters, n > 0)
	n += copeN
	return
}

func CopyPacketWithPool(originSource N.PacketReader, destinationConn N.PacketWriter, source N.PacketReader, readCounters []N.CountFunc, writeCounters []N.CountFunc, notFirstTime bool) (n int64, err error) {
	frontHeadroom := N.CalculateFrontHeadroom(destinationConn)
	rearHeadroom := N.CalculateRearHeadroom(destinationConn)
	bufferSize := N.CalculateMTU(source, destinationConn)
	if bufferSize > 0 {
		bufferSize += frontHeadroom + rearHeadroom
	} else {
		bufferSize = buf.UDPBufferSize
	}
	var destination M.Socksaddr
	for {
		buffer := buf.NewSize(bufferSize)
		buffer.Resize(frontHeadroom, 0)
		buffer.Reserve(rearHeadroom)
		destination, err = source.ReadPacket(buffer)
		if err != nil {
			buffer.Release()
			return
		}
		dataLen := buffer.Len()
		buffer.OverCap(rearHeadroom)
		err = destinationConn.WritePacket(buffer, destination)
		if err != nil {
			buffer.Leak()
			if !notFirstTime {
				err = N.ReportHandshakeFailure(originSource, err)
			}
			return
		}
		n += int64(dataLen)
		for _, counter := range readCounters {
			counter(int64(dataLen))
		}
		for _, counter := range writeCounters {
			counter(int64(dataLen))
		}
		notFirstTime = true
	}
}

func WritePacketWithPool(originSource N.PacketReader, destinationConn N.PacketWriter, packetBuffers []*N.PacketBuffer) (n int64, err error) {
	frontHeadroom := N.CalculateFrontHeadroom(destinationConn)
	rearHeadroom := N.CalculateRearHeadroom(destinationConn)
	var notFirstTime bool
	for _, packetBuffer := range packetBuffers {
		buffer := buf.NewPacket()
		buffer.Resize(frontHeadroom, 0)
		buffer.Reserve(rearHeadroom)
		_, err = buffer.Write(packetBuffer.Buffer.Bytes())
		packetBuffer.Buffer.Release()
		if err != nil {
			buffer.Release()
			continue
		}
		dataLen := buffer.Len()
		buffer.OverCap(rearHeadroom)
		err = destinationConn.WritePacket(buffer, packetBuffer.Destination)
		if err != nil {
			buffer.Leak()
			if !notFirstTime {
				err = N.ReportHandshakeFailure(originSource, err)
			}
			return
		}
		n += int64(dataLen)
	}
	return
}

func CopyPacketConn(ctx context.Context, source N.PacketConn, destination N.PacketConn) error {
	return CopyPacketConnContextList([]context.Context{ctx}, source, destination)
}

func CopyPacketConnContextList(contextList []context.Context, source N.PacketConn, destination N.PacketConn) error {
	wg := sync.WaitGroup{}
	wg.Add(2)
	var err1, err2 error

	go func() {
		defer wg.Done()
		defer source.Close()
		defer destination.Close()
		err1 = common.Error(CopyPacket(destination, source))
		if err1 != nil {
			err1 = E.Cause(err1, "upload")
		}
	}()
	go func() {
		defer wg.Done()
		defer source.Close()
		defer destination.Close()
		err2 = common.Error(CopyPacket(source, destination))
		if err2 != nil {
			err2 = E.Cause(err2, "download")
		}
	}()
	wg.Wait()

	var retErr error
	if err1 != nil {
		retErr = E.Errors(err1)
	}
	if err2 != nil {
		retErr = E.Errors(err2)
	}
	return retErr
}
