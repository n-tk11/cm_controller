FROM ffdev:withssl

COPY /home/tul/MyProj/fastfreeze fastfreeze
COPY /home/tul/MyProj/run.json run.json
COPY /home/tul/MyProj/chk.json chk.json

ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH
ENV RUSTUP_HOME=/opt/rust CARGO_HOME=/opt/cargo PATH=/opt/cargo/bin:/usr/local/bin:$PATH

RUN apt update && apt install vim python3 golang-go -y
RUN cd fastfreeze && make && ln -s /opt/fastfreeze/fastfreeze /usr/local/bin  && fastfreeze install

RUN setcap cap_sys_ptrace+eip /opt/fastfreeze/bin/criu
RUN setcap 40+eip /opt/fastfreeze/bin/set_ns_last_pid

#Build ff_daemon
COPY /home/tul/MyProj/ff_daemon ff_daemon
RUN cd ff_daemon && cargo build && ln -s /ff_daemon/target/debug/ff_daemon /usr/local/bin

#For metric testing
#COPY metric.c metric.c
#COPY run_m.json run_m.json
#RUN gcc -o metric metric.c

#Test decider
COPY /home/tul/MyProj/decider.txt decider.txt
COPY /home/tul/MyProj/decider_logic.py decider_logic.py
RUN chmod 770 decider_logic.py 


