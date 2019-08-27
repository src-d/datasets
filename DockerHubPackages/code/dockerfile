FROM ubuntu:16.04

RUN apt update
RUN apt install bc nodejs npm ipython3 git docker.io
RUN npm i -g npm
RUN git clone https://github.com/glimow/docker_image_analysis.git
RUN git clone https://github.com/glimow/cli.git
RUN (cd cli npm install)
ENV NPM_PATCH_PATH /cli
ENTRYPOINT ipython /images.txt /packages 0 1