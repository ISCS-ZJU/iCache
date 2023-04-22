# Generated by the gRPC Python protocol compiler plugin. DO NOT EDIT!
"""Client and server classes corresponding to protobuf-defined services."""
import grpc

import deepcache_pb2 as deepcache__pb2


class OperatorStub(object):
    """Missing associated documentation comment in .proto file."""

    def __init__(self, channel):
        """Constructor.

        Args:
            channel: A grpc.Channel.
        """
        self.DCSubmit = channel.unary_unary(
                '/rpc.Operator/DCSubmit',
                request_serializer=deepcache__pb2.DCRequest.SerializeToString,
                response_deserializer=deepcache__pb2.DCReply.FromString,
                )


class OperatorServicer(object):
    """Missing associated documentation comment in .proto file."""

    def DCSubmit(self, request, context):
        """Submit a request
        """
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details('Method not implemented!')
        raise NotImplementedError('Method not implemented!')


def add_OperatorServicer_to_server(servicer, server):
    rpc_method_handlers = {
            'DCSubmit': grpc.unary_unary_rpc_method_handler(
                    servicer.DCSubmit,
                    request_deserializer=deepcache__pb2.DCRequest.FromString,
                    response_serializer=deepcache__pb2.DCReply.SerializeToString,
            ),
    }
    generic_handler = grpc.method_handlers_generic_handler(
            'rpc.Operator', rpc_method_handlers)
    server.add_generic_rpc_handlers((generic_handler,))


 # This class is part of an EXPERIMENTAL API.
class Operator(object):
    """Missing associated documentation comment in .proto file."""

    @staticmethod
    def DCSubmit(request,
            target,
            options=(),
            channel_credentials=None,
            call_credentials=None,
            insecure=False,
            compression=None,
            wait_for_ready=None,
            timeout=None,
            metadata=None):
        return grpc.experimental.unary_unary(request, target, '/rpc.Operator/DCSubmit',
            deepcache__pb2.DCRequest.SerializeToString,
            deepcache__pb2.DCReply.FromString,
            options, channel_credentials,
            insecure, call_credentials, compression, wait_for_ready, timeout, metadata)
